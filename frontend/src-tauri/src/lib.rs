mod sidecar;

use tauri::Manager;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let application = tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .manage(sidecar::SidecarState::default())
        .invoke_handler(tauri::generate_handler![
            sidecar::sidecar_start,
            sidecar::sidecar_write,
            sidecar::sidecar_stop,
        ])
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                match window.label() {
                    "storage-analyzer" => {
                        api.prevent_close();
                        let _ = window.hide();
                    }
                    "main" if sidecar::begin_graceful_exit(window.app_handle()) => {
                        api.prevent_close();
                    }
                    _ => {}
                }
            }
        })
        .build(tauri::generate_context!())
        .expect("failed to build Luxury Optimization desktop application");

    application.run(|app, event| match event {
        tauri::RunEvent::ExitRequested { api, .. } => {
            if sidecar::begin_graceful_exit(app) {
                api.prevent_exit();
            }
        }
        tauri::RunEvent::Exit => sidecar::stop_on_exit(app),
        _ => {}
    });
}
