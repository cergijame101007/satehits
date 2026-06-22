const API_BASE = import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080';

interface ErrorResponse {
  error: {
    code: string;
    message: string;
    details?: Array<{ field: string; message: string }>;
  };
}

interface PublicSupplierResponse {
  id: number;
  name: string;
  description: string;
  instagram_url?: string | null;
  image_url?: string | null;
}

interface PublicSupplierListResponse {
  suppliers: PublicSupplierResponse[];
}

export interface PublicSupplier {
  id: number;
  name: string;
  description: string;
  instagram_url: string | null;
  image_url: string | null;
}

export class PublicSupplierApiError extends Error {
  code: string;
  details?: Array<{ field: string; message: string }>;

  constructor(error: ErrorResponse['error']) {
    super(error.message);
    this.name = 'PublicSupplierApiError';
    this.code = error.code;
    this.details = error.details;
  }
}

export async function parsePublicSupplierError(res: Response): Promise<PublicSupplierApiError> {
  try {
    const body = (await res.json()) as ErrorResponse;
    if (body?.error?.message) {
      return new PublicSupplierApiError(body.error);
    }
  } catch {
    // fall through
  }
  return new PublicSupplierApiError({
    code: 'INTERNAL_ERROR',
    message: 'サーバー内部でエラーが発生しました',
  });
}

/** API レスポンスを公開ページ用型に正規化する */
export function normalizePublicSupplier(item: PublicSupplierResponse): PublicSupplier {
  return {
    id: item.id,
    name: item.name,
    description: item.description,
    instagram_url: item.instagram_url ?? null,
    image_url: item.image_url ?? null,
  };
}

/** GET /api/v1/suppliers */
export async function listPublicSuppliers(): Promise<PublicSupplier[]> {
  const res = await fetch(`${API_BASE}/api/v1/suppliers`);

  if (!res.ok) {
    throw await parsePublicSupplierError(res);
  }

  const data = (await res.json()) as PublicSupplierListResponse;
  return data.suppliers.map(normalizePublicSupplier);
}
