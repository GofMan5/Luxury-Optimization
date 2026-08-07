use std::{
    sync::{
        Mutex, MutexGuard,
        atomic::{AtomicBool, AtomicU64, Ordering},
    },
    thread,
    time::Duration,
};

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter, Manager, State};
use tauri_plugin_shell::{
    ShellExt,
    process::{CommandChild, CommandEvent},
};

const MAX_REQUEST_BYTES: usize = 256 * 1024;
const MAX_RESULT_BYTES: usize = 1024 * 1024;
const FRAME_EVENT: &str = "sidecar-frame";
const LIFECYCLE_EVENT: &str = "sidecar-lifecycle";

#[derive(Default)]
pub struct SidecarState {
    child: Mutex<Option<CommandChild>>,
    generation: AtomicU64,
    exiting: AtomicBool,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StartResult {
    pid: u32,
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct LifecycleEvent {
    state: &'static str,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct OutgoingEnvelope {
    v: u8,
    id: String,
    #[serde(rename = "type")]
    kind: String,
    method: String,
    #[serde(default, rename = "payload")]
    _payload: Option<serde_json::Value>,
}

#[tauri::command]
pub fn sidecar_start(
    app: AppHandle,
    state: State<'_, SidecarState>,
) -> Result<StartResult, String> {
    let mut guard = lock_start(state.inner())?;
    if let Some(child) = guard.as_ref() {
        return Ok(StartResult { pid: child.pid() });
    }

    let (receiver, child) = app
        .shell()
        .sidecar("luxury-optimization-backend")
        .map_err(|_| "Optimization backend is unavailable".to_string())?
        .set_raw_out(true)
        .spawn()
        .map_err(|_| "Optimization backend could not start".to_string())?;
    let pid = child.pid();
    let generation = state.generation.fetch_add(1, Ordering::AcqRel) + 1;
    *guard = Some(child);
    drop(guard);

    let event_app = app.clone();
    tauri::async_runtime::spawn(async move {
        forward_events(event_app, receiver, pid, generation).await;
    });
    Ok(StartResult { pid })
}

fn lock_start(state: &SidecarState) -> Result<MutexGuard<'_, Option<CommandChild>>, String> {
    let guard = state
        .child
        .lock()
        .map_err(|_| "Optimization backend state is unavailable".to_string())?;
    if state.exiting.load(Ordering::Acquire) {
        return Err("Application is exiting".to_string());
    }
    Ok(guard)
}

#[tauri::command]
pub fn sidecar_write(state: State<'_, SidecarState>, frame: String) -> Result<(), String> {
    validate_outgoing(&frame)?;
    let mut guard = state
        .child
        .lock()
        .map_err(|_| "Optimization backend state is unavailable".to_string())?;
    let child = guard
        .as_mut()
        .ok_or_else(|| "Optimization backend is not running".to_string())?;
    child
        .write(format!("{frame}\n").as_bytes())
        .map_err(|_| "Optimization command could not be sent".to_string())
}

#[tauri::command]
pub fn sidecar_stop(state: State<'_, SidecarState>) -> Result<(), String> {
    let child = state
        .child
        .lock()
        .map_err(|_| "Optimization backend state is unavailable".to_string())?
        .take();
    if let Some(child) = child {
        child
            .kill()
            .map_err(|_| "Optimization backend could not be stopped".to_string())?;
    }
    Ok(())
}

pub fn stop_on_exit(app: &AppHandle) {
    if let Ok(mut guard) = app.state::<SidecarState>().child.lock()
        && let Some(child) = guard.take()
    {
        let _ = child.kill();
    }
}

pub fn begin_graceful_exit(app: &AppHandle) -> bool {
    let state = app.state::<SidecarState>();
    if state.exiting.swap(true, Ordering::AcqRel) {
        return false;
    }
    if let Ok(mut guard) = state.child.lock()
        && let Some(child) = guard.as_mut()
    {
        let _ = child.write(
            b"{\"v\":1,\"id\":\"native_shutdown\",\"type\":\"command\",\"method\":\"system.shutdown\",\"payload\":{}}\n",
        );
    }
    let app = app.clone();
    thread::spawn(move || {
        for _ in 0..100 {
            if app
                .state::<SidecarState>()
                .child
                .lock()
                .map(|child| child.is_none())
                .unwrap_or(true)
            {
                app.exit(0);
                return;
            }
            thread::sleep(Duration::from_millis(50));
        }
        stop_on_exit(&app);
        app.exit(0);
    });
    true
}

fn validate_outgoing(frame: &str) -> Result<(), String> {
    if frame.is_empty()
        || frame.len() > MAX_REQUEST_BYTES
        || frame
            .bytes()
            .any(|byte| matches!(byte, b'\r' | b'\n' | b'\0'))
    {
        return Err("Invalid optimization protocol frame".to_string());
    }
    let envelope: OutgoingEnvelope = serde_json::from_str(frame)
        .map_err(|_| "Invalid optimization protocol frame".to_string())?;
    if envelope.v != 1
        || envelope.kind != "command"
        || !valid_request_id(&envelope.id)
        || !allowed_method(&envelope.method)
    {
        return Err("Optimization command is not allowed".to_string());
    }
    Ok(())
}

fn valid_request_id(value: &str) -> bool {
    !value.is_empty()
        && value.len() <= 80
        && value
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || matches!(byte, b'_' | b'-'))
}

fn allowed_method(method: &str) -> bool {
    matches!(
        method,
        "system.handshake"
            | "system.cancel"
            | "system.shutdown"
            | "optimization.audit"
            | "optimization.plan"
            | "optimization.apply"
            | "optimization.restore"
            | "optimization.apply_tweak"
            | "optimization.restore_tweak"
            | "optimization.checkpoint_status"
            | "optimization.create_checkpoint"
            | "backups.list"
            | "restore.system_points"
            | "restore.open_system"
            | "startup.list"
            | "startup.set"
            | "services.list"
            | "services.set"
            | "network.interfaces"
            | "network.test"
            | "benchmark.compare"
            | "gaming.scan"
            | "gaming.saved"
            | "gaming.save"
            | "gaming.remove"
            | "gaming.launch"
            | "gaming.history"
            | "gaming.attach_benchmark"
            | "cleanup.run"
            | "updates.status"
    )
}

async fn forward_events(
    app: AppHandle,
    mut receiver: tauri::async_runtime::Receiver<CommandEvent>,
    pid: u32,
    generation: u64,
) {
    let mut buffer = Vec::with_capacity(8 * 1024);
    let mut lifecycle_state = "stopped";
    while let Some(event) = receiver.recv().await {
        match event {
            CommandEvent::Stdout(bytes) => {
                if append_frames(&app, &mut buffer, &bytes).is_err() {
                    lifecycle_state = "protocol-error";
                    break;
                }
            }
            CommandEvent::Terminated(_) => break,
            CommandEvent::Error(_) => {
                lifecycle_state = "error";
                break;
            }
            CommandEvent::Stderr(_) => {
                // Backend diagnostics never cross into the WebView.
            }
            _ => {}
        }
    }
    let state = app.state::<SidecarState>();
    let mut lifecycle = None;
    if let Ok(mut child) = state.child.lock() {
        let active_generation = state.generation.load(Ordering::Acquire);
        lifecycle = lifecycle_for_process(
            child.as_ref().map(CommandChild::pid),
            active_generation,
            pid,
            generation,
            lifecycle_state,
        );
        if lifecycle.is_some()
            && let Some(child) = child.take()
        {
            let _ = child.kill();
        }
    }
    if let Some(state) = lifecycle {
        let _ = app.emit(LIFECYCLE_EVENT, LifecycleEvent { state });
    }
}

fn lifecycle_for_process(
    active_pid: Option<u32>,
    active_generation: u64,
    completed_pid: u32,
    completed_generation: u64,
    state: &'static str,
) -> Option<&'static str> {
    matching_process(
        active_pid,
        active_generation,
        completed_pid,
        completed_generation,
    )
    .then_some(state)
}

fn matching_process(
    active_pid: Option<u32>,
    active_generation: u64,
    completed_pid: u32,
    completed_generation: u64,
) -> bool {
    active_pid == Some(completed_pid) && active_generation == completed_generation
}

fn append_frames(app: &AppHandle, buffer: &mut Vec<u8>, bytes: &[u8]) -> Result<(), ()> {
    decode_frames(buffer, bytes, |text| {
        app.emit(FRAME_EVENT, text).map_err(|_| ())
    })
}

fn decode_frames(
    buffer: &mut Vec<u8>,
    bytes: &[u8],
    mut emit: impl FnMut(String) -> Result<(), ()>,
) -> Result<(), ()> {
    for &byte in bytes {
        if byte != b'\n' {
            if buffer.len() == MAX_RESULT_BYTES {
                return Err(());
            }
            buffer.push(byte);
            continue;
        }
        if buffer.last() == Some(&b'\r') {
            buffer.pop();
        }
        if buffer.is_empty() {
            return Err(());
        }
        let frame = std::mem::replace(buffer, Vec::with_capacity(8 * 1024));
        let text = String::from_utf8(frame).map_err(|_| ())?;
        let value: serde_json::Value = serde_json::from_str(&text).map_err(|_| ())?;
        if value.get("v").and_then(serde_json::Value::as_u64) != Some(1)
            || value.get("type").and_then(serde_json::Value::as_str) != Some("result")
            || !value
                .get("id")
                .and_then(serde_json::Value::as_str)
                .is_some_and(valid_request_id)
        {
            return Err(());
        }
        emit(text)?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_newlines_bad_ids_and_unlisted_methods() {
        assert!(validate_outgoing("{}\n").is_err());
        assert!(
            validate_outgoing(r#"{"v":1,"id":"x","type":"command","method":"shell.run"}"#).is_err()
        );
        assert!(
            validate_outgoing(
                r#"{"v":1,"id":"bad id","type":"command","method":"optimization.audit"}"#
            )
            .is_err()
        );
        assert!(
            validate_outgoing(
                r#"{"v":1,"id":"x","type":"command","method":"updates.install","payload":{}}"#
            )
            .is_err()
        );
    }

    #[test]
    fn accepts_allowlisted_commands() {
        for method in [
            "system.handshake",
            "optimization.audit",
            "optimization.apply_tweak",
            "optimization.restore_tweak",
            "services.list",
            "benchmark.compare",
            "gaming.attach_benchmark",
            "updates.status",
        ] {
            let frame = format!(
                r#"{{"v":1,"id":"x","type":"command","method":"{method}","payload":{{}}}}"#
            );
            assert!(validate_outgoing(&frame).is_ok());
        }
    }

    #[test]
    fn decodes_multiple_bounded_result_frames() {
        let frame = b"{\"v\":1,\"id\":\"x\",\"type\":\"result\",\"ok\":true}\n";
        let bytes = frame.repeat(20);
        let mut buffer = Vec::new();
        let mut count = 0;
        decode_frames(&mut buffer, &bytes, |_| {
            count += 1;
            Ok(())
        })
        .unwrap();
        assert_eq!(count, 20);
        assert!(buffer.is_empty());
    }

    #[test]
    fn stale_reader_cannot_claim_restarted_sidecar() {
        assert!(matching_process(Some(10), 1, 10, 1));
        assert!(!matching_process(Some(10), 2, 10, 1));
        assert!(!matching_process(Some(11), 1, 10, 1));
        assert_eq!(
            lifecycle_for_process(Some(10), 1, 10, 1, "protocol-error"),
            Some("protocol-error")
        );
    }
}
