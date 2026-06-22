import { describe, expect, it } from 'vitest';
import { normalizePublicSupplier } from '@/lib/publicSuppliers';
import { normalizeSupplier, validateSupplierImageFile } from '@/lib/suppliers';

describe('suppliers lib', () => {
  it('normalizes admin supplier response with nullable fields', () => {
    const supplier = normalizeSupplier({
      id: 1,
      name: 'テスト農園',
      description: '説明',
      display_order: 2,
      is_active: true,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-02T00:00:00Z',
    });

    expect(supplier.instagram_url).toBeNull();
    expect(supplier.image_url).toBeNull();
    expect(supplier.display_order).toBe(2);
  });

  it('accepts allowed image types under size limit', () => {
    const file = new File(['x'], 'photo.jpg', { type: 'image/jpeg' });
    Object.defineProperty(file, 'size', { value: 1024 });

    expect(validateSupplierImageFile(file)).toBeNull();
  });

  it('rejects unsupported image types', () => {
    const file = new File(['x'], 'photo.gif', { type: 'image/gif' });
    Object.defineProperty(file, 'size', { value: 1024 });

    expect(validateSupplierImageFile(file)).toContain('JPEG / PNG / WebP');
  });
});

describe('publicSuppliers lib', () => {
  it('normalizes public supplier response', () => {
    const supplier = normalizePublicSupplier({
      id: 3,
      name: '公開農園',
      description: '公開説明',
      instagram_url: 'https://instagram.com/example',
      image_url: 'http://localhost:9000/satehits-images/suppliers/3/a.jpg',
    });

    expect(supplier.instagram_url).toBe('https://instagram.com/example');
    expect(supplier.image_url).toContain('suppliers/3/a.jpg');
  });
});
