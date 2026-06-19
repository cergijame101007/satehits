import { cx } from '@/lib/cx';
import type { ButtonSize, ButtonVariant } from '@/lib/ui/types';

const solidDisabledClass =
  'disabled:bg-gray-400 disabled:text-white disabled:opacity-100 disabled:shadow-none disabled:hover:bg-gray-400 disabled:active:scale-100';

const baseClass =
  'inline-flex items-center justify-center rounded-lg font-medium transition-colors disabled:cursor-not-allowed';

export const variantClasses: Record<ButtonVariant, string> = {
  primary: cx(
    'bg-primary text-white hover:bg-primary-dark active:scale-[0.98] shadow-lg',
    solidDisabledClass,
  ),
  success: cx('bg-green-600 text-white hover:bg-green-700', solidDisabledClass),
  danger: cx('bg-red-600 text-white hover:bg-red-700', solidDisabledClass),
  warning: cx('bg-orange-600 text-white hover:bg-orange-700', solidDisabledClass),
  neutral: cx('bg-gray-700 text-white hover:bg-gray-800', solidDisabledClass),
  ghost:
    'border border-gray-300 text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:hover:bg-white disabled:hover:border-gray-300',
};

export const sizeClasses: Record<ButtonSize, string> = {
  sm: 'px-4 py-1.5 text-sm',
  md: 'px-4 py-2 text-sm',
  lg: 'w-full py-3 rounded-xl text-base',
};

/** Button / ButtonLink 共通の className */
export function getButtonClassName(
  variant: ButtonVariant = 'primary',
  size: ButtonSize = 'md',
  fullWidth = false,
): string {
  return cx(
    baseClass,
    variantClasses[variant],
    sizeClasses[size],
    fullWidth && size !== 'lg' && 'w-full',
  );
}
