import { forwardRef, type ButtonHTMLAttributes, type PropsWithChildren } from 'react'

type ButtonVariant = 'primary' | 'secondary' | 'quiet' | 'danger'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
}

export const Button = forwardRef<HTMLButtonElement, PropsWithChildren<ButtonProps>>(function Button({ variant = 'secondary', className = '', children, ...props }, ref) {
  return <button ref={ref} className={`button button--${variant} ${className}`} {...props}>{children}</button>
})
