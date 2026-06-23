// Accent primary + ghost variants. No shadows, no gradients — borders only.
import { forwardRef } from 'react'
import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { cn } from '../../utils/cn'

type Variant = 'primary' | 'ghost' | 'danger' | 'subtle'
type Size = 'sm' | 'md' | 'lg'

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  loading?: boolean
  iconLeft?: ReactNode
  iconRight?: ReactNode
}

const sizes: Record<Size, string> = {
  sm: 'h-8 px-3 text-sm',
  md: 'h-10 px-4 text-sm',
  lg: 'h-12 px-6 text-base',
}

const variants: Record<Variant, string> = {
  primary: 'bg-accent text-accent-fg border border-accent hover:bg-accent-hover hover:border-accent-hover active:bg-accent-pressed',
  ghost: 'bg-transparent text-fg border border-border hover:bg-bg-sunken hover:border-border-strong',
  subtle: 'bg-bg-sunken text-fg border border-transparent hover:bg-bg-elevated hover:border-border',
  danger: 'bg-transparent text-verdict-false border border-verdict-false hover:bg-verdict-false-bg',
}

export const Button = forwardRef<HTMLButtonElement, Props>(function Button(
  { variant = 'ghost', size = 'md', loading, iconLeft, iconRight, className, children, disabled, ...rest },
  ref,
) {
  return (
    <button
      ref={ref}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cn(
        'inline-flex items-center justify-center gap-2 rounded font-medium',
        'transition-colors duration-micro ease-in-out-quart',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        sizes[size],
        variants[variant],
        className,
      )}
      {...rest}
    >
      {iconLeft && <span aria-hidden="true">{iconLeft}</span>}
      <span>{children}</span>
      {loading && <span className="sr-only">Loading…</span>}
      {loading && <span aria-hidden="true" className="ml-1">…</span>}
      {iconRight && <span aria-hidden="true">{iconRight}</span>}
    </button>
  )
})
