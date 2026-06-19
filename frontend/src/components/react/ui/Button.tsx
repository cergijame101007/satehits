import type { ButtonHTMLAttributes, ReactNode } from 'react';
import { cx } from '@/lib/cx';
import type { ButtonSize, ButtonVariant } from '@/components/react/ui/types';

export type { ButtonSize, ButtonVariant } from '@/components/react/ui/types';

const baseClass = 'rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed';

const variantClasses: Record<ButtonVariant, string> = {
  primary: 'bg-primary text-white hover:bg-primary-dark active:scale-[0.98] shadow-lg',
  success: 'bg-green-600 text-white hover:bg-green-700',
  danger: 'bg-red-600 text-white hover:bg-red-700',
  warning: 'bg-orange-600 text-white hover:bg-orange-700',
  neutral: 'bg-gray-700 text-white hover:bg-gray-800',
  ghost: 'border border-gray-300 text-gray-700 bg-white hover:bg-gray-50',
};

const sizeClasses: Record<ButtonSize, string> = {
  sm: 'px-4 py-1.5 text-sm',
  md: 'px-4 py-2 text-sm',
  lg: 'w-full py-3 rounded-xl text-base',
};

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  children: ReactNode;
}

export default function Button({
  variant = 'primary',
  size = 'md',
  fullWidth = false,
  className,
  type = 'button',
  children,
  ...props
}: ButtonProps) {
  return (
    <button
      type={type}
      className={cx(
        baseClass,
        variantClasses[variant],
        sizeClasses[size],
        fullWidth && size !== 'lg' && 'w-full',
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}
