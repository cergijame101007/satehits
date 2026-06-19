import { cx } from '@/lib/cx';

const baseInputClass =
  'w-full rounded-lg border bg-white focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-colors';

/** フォーム input / select / textarea 共通スタイル */
export function inputClassName(hasError = false, size: 'base' | 'sm' = 'base'): string {
  return cx(
    baseInputClass,
    size === 'base' ? 'px-4 py-3 text-base' : 'px-3 py-2 text-sm',
    hasError ? 'border-red-400 bg-red-50/50' : 'border-gray-200',
  );
}
