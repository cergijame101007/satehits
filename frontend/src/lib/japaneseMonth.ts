/** 和暦風の月名（1=睦月 … 12=師走） */
const JAPANESE_MONTH_NAMES = [
  '睦月',
  '如月',
  '弥生',
  '卯月',
  '皐月',
  '水無月',
  '文月',
  '葉月',
  '長月',
  '神無月',
  '霜月',
  '師走',
] as const;

export function getJapaneseMonthName(month: number): string {
  if (month < 1 || month > 12) return '';
  return JAPANESE_MONTH_NAMES[month - 1];
}
