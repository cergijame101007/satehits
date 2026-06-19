import type { ReactNode } from 'react';
import { cx } from '@/lib/cx';

export type AlertVariant = 'error' | 'warning' | 'info';

const variantClasses: Record<AlertVariant, string> = {
  error: 'rounded-lg bg-red-50 border border-red-200 text-red-700',
  warning: 'rounded-lg bg-amber-50 border border-amber-200 text-amber-800',
  info: 'rounded-lg bg-primary/5 border border-primary/10 text-gray-700',
};

export interface AlertProps {
  variant?: AlertVariant;
  compact?: boolean;
  className?: string;
  children: ReactNode;
}

export default function Alert({
  variant = 'error',
  compact = false,
  className,
  children,
}: AlertProps) {
  return (
    <div
      role="alert"
      className={cx(variantClasses[variant], compact ? 'p-2 text-xs' : 'p-3 text-sm', className)}
    >
      {children}
    </div>
  );
}
