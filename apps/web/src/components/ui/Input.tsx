// Floating-label input with the accent-on-focus signature motion.
import { forwardRef, useId, useState } from 'react'
import type { InputHTMLAttributes, ReactNode } from 'react'
import { cn } from '../../utils/cn'

interface Props extends Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> {
  label: string
  description?: string
  error?: string
  iconRight?: ReactNode
}

export const Input = forwardRef<HTMLInputElement, Props>(function Input(
  { label, description, error, iconRight, className, id, onFocus, onBlur, value, defaultValue, ...rest },
  ref,
) {
  const generatedId = useId()
  const inputId = id ?? generatedId
  const [focused, setFocused] = useState(false)
  const hasValue = (value !== undefined && value !== '') || (defaultValue !== undefined && defaultValue !== '')
  const floating = focused || hasValue

  return (
    <div className="w-full">
      <div
        className={cn(
          'relative w-full rounded-md border bg-bg-elevated transition-colors duration-micro ease-in-out-quart',
          focused ? 'border-accent border-2 bg-accent-bg' : 'border-border',
          error && 'border-verdict-false',
        )}
      >
        <label
          htmlFor={inputId}
          className={cn(
            'pointer-events-none absolute left-4 transition-all duration-macro ease-in-out-quart',
            floating
              ? 'top-1 text-2xs uppercase tracking-wide font-medium text-fg-muted'
              : 'top-1/2 -translate-y-1/2 text-base text-fg-muted',
          )}
        >
          {label}
        </label>
        <input
          ref={ref}
          id={inputId}
          value={value}
          defaultValue={defaultValue}
          onFocus={(e) => { setFocused(true); onFocus?.(e) }}
          onBlur={(e) => { setFocused(false); onBlur?.(e) }}
          className={cn(
            'w-full bg-transparent text-fg-strong placeholder:text-fg-muted/0',
            'px-4 pt-5 pb-2 text-base outline-none',
            // adjust for the 2px border state
            focused ? 'px-[15px] pt-[19px] pb-[7px]' : '',
            iconRight ? 'pr-12' : '',
            className,
          )}
          aria-describedby={error ? `${inputId}-err` : description ? `${inputId}-desc` : undefined}
          aria-invalid={!!error}
          {...rest}
        />
        {iconRight && (
          <div className="absolute right-3 top-1/2 -translate-y-1/2 text-fg-muted" aria-hidden="true">
            {iconRight}
          </div>
        )}
      </div>
      {description && !error && (
        <p id={`${inputId}-desc`} className="mt-2 text-sm text-fg-muted">{description}</p>
      )}
      {error && (
        <p id={`${inputId}-err`} role="alert" className="mt-2 text-sm text-verdict-false">{error}</p>
      )}
    </div>
  )
})
