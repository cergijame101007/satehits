/** カレンダーヘッダー表示用の店舗定例（暫定） */

// NOTE: 店舗定例（営業時間・定休日）は現状 domain_knowledge / フロント定数。
// 今後オーナーが店舗定例を設定できる機能追加時は、ここを API 取得に差し替える。
export const STORE_HOURS_HEADER = {
  regularOpen: '11:30',
  regularClose: '15:00',
  sundayOpen: '8:30',
  sundayClose: '15:00',
  lastOrder: '13:30',
  closedWeekdaysLabel: '木、金定休日',
} as const;

export function getStoreHoursHeaderContent(): {
  openLabel: string;
  hoursLines: string[];
  closedWeekdaysLabel: string;
} {
  const { regularOpen, regularClose, sundayOpen, sundayClose, lastOrder, closedWeekdaysLabel } =
    STORE_HOURS_HEADER;

  return {
    openLabel: 'open',
    hoursLines: [
      `${regularOpen}-${regularClose}`,
      `日 ${sundayOpen}-${sundayClose}`,
      `(L.O ${lastOrder})`,
    ],
    closedWeekdaysLabel,
  };
}
