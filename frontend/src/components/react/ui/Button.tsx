import type { ButtonHTMLAttributes, ReactNode } from 'react';
import { getButtonClassName } from '@/lib/ui/buttonStyles';
import type { ButtonSize, ButtonVariant } from '@/lib/ui/types';

export type { ButtonSize, ButtonVariant } from '@/lib/ui/types';

export interface ButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'className'> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  children: ReactNode;
}

export default function Button({
  variant = 'primary',
  size = 'md',
  fullWidth = false,
  type = 'button',
  children,
  ...props
}: ButtonProps) {
  return (
    <button type={type} className={getButtonClassName(variant, size, fullWidth)} {...props}>
      {children}
    </button>
  );
}
