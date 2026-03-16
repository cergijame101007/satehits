import type { Supplier } from '../types/supplier';

/** モック取引先データ */
export const mockSuppliers: Supplier[] = [
  {
    id: 1,
    name: '〇〇農園',
    description:
      '富山県で無農薬野菜を栽培されています。旬の野菜を直接仕入れています。',
    instagram_url: 'https://www.instagram.com/example_farm/',
    image_url: null,
    display_order: 1,
    is_active: true,
    created_at: '2026-01-15T10:00:00+09:00',
    updated_at: '2026-01-15T10:00:00+09:00',
  },
  {
    id: 2,
    name: '△△和紅茶園',
    description:
      '国産和紅茶を生産されています。当店の和紅茶はこちらから仕入れています。',
    instagram_url: 'https://www.instagram.com/example_tea/',
    image_url: null,
    display_order: 2,
    is_active: true,
    created_at: '2026-01-15T10:00:00+09:00',
    updated_at: '2026-01-15T10:00:00+09:00',
  },
  {
    id: 3,
    name: '□□水産',
    description:
      '富山湾の新鮮な魚を毎朝届けていただいています。',
    instagram_url: 'https://www.instagram.com/example_fish/',
    image_url: null,
    display_order: 3,
    is_active: false,
    created_at: '2026-01-20T10:00:00+09:00',
    updated_at: '2026-02-01T10:00:00+09:00',
  },
  {
    id: 4,
    name: '◎◎養蜂場',
    description:
      '富山県産の天然はちみつを生産されています。季節のデザートに使用しています。',
    instagram_url: 'https://www.instagram.com/example_honey/',
    image_url: null,
    display_order: 4,
    is_active: true,
    created_at: '2026-02-10T10:00:00+09:00',
    updated_at: '2026-02-10T10:00:00+09:00',
  },
];

/**
 * 取引先一覧を取得（モック）
 * TODO: GET /api/v1/admin/suppliers に置き換え
 */
export function getSuppliers(activeOnly = false): Supplier[] {
  let suppliers = [...mockSuppliers].sort(
    (a, b) => a.display_order - b.display_order,
  );
  if (activeOnly) {
    suppliers = suppliers.filter((s) => s.is_active);
  }
  return suppliers;
}

/**
 * 取引先を取得（モック）
 * TODO: GET /api/v1/admin/suppliers/:id に置き換え
 */
export function getSupplier(id: number): Supplier | undefined {
  return mockSuppliers.find((s) => s.id === id);
}
