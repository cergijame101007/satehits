import type { ReactNode } from 'react';
import { cx } from '@/lib/cx';

export interface CardProps {
  children: ReactNode;
  compact?: boolean;
  className?: string;
}

export default function Card({ children, compact = false, className }: CardProps) {
  return (
    <div
      className={cx(
        'rounded-xl border border-gray-200 bg-white',
        compact ? 'p-4' : 'p-5',
        className,
      )}
    >
      {children}
    </div>
  );
}
