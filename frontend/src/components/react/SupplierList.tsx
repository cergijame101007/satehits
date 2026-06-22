import { useEffect, useState } from 'react';
import type { PublicSupplier } from '@/lib/publicSuppliers';
import { listPublicSuppliers, PublicSupplierApiError } from '@/lib/publicSuppliers';

function ImagePlaceholder() {
  return (
    <div className="flex h-48 w-full items-center justify-center bg-gradient-to-br from-gray-100 to-gray-50">
      <svg className="h-12 w-12 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
        />
      </svg>
    </div>
  );
}

function SupplierCard({ supplier }: { supplier: PublicSupplier }) {
  return (
    <div className="overflow-hidden rounded-xl border border-gray-200 bg-white">
      {supplier.image_url ? (
        <img
          src={supplier.image_url}
          alt={supplier.name}
          className="h-48 w-full object-cover"
          loading="lazy"
        />
      ) : (
        <ImagePlaceholder />
      )}

      <div className="space-y-3 p-5">
        <h2 className="text-lg font-medium">{supplier.name}</h2>
        <p className="text-sm leading-relaxed text-gray-600">{supplier.description}</p>

        {supplier.instagram_url && (
          <a
            href={supplier.instagram_url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 text-sm text-primary transition-colors hover:text-primary-dark"
          >
            <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zM12 0C8.741 0 8.333.014 7.053.072 2.695.272.273 2.69.073 7.052.014 8.333 0 8.741 0 12c0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98C8.333 23.986 8.741 24 12 24c3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98C15.668.014 15.259 0 12 0zm0 5.838a6.162 6.162 0 100 12.324 6.162 6.162 0 000-12.324zM12 16a4 4 0 110-8 4 4 0 010 8zm6.406-11.845a1.44 1.44 0 100 2.881 1.44 1.44 0 000-2.881z" />
            </svg>
            Instagramを見る
          </a>
        )}
      </div>
    </div>
  );
}

export default function SupplierList() {
  const [suppliers, setSuppliers] = useState<PublicSupplier[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState('');

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setIsLoading(true);
      setLoadError('');

      try {
        const data = await listPublicSuppliers();
        if (!cancelled) {
          setSuppliers(data);
        }
      } catch (err) {
        if (!cancelled) {
          setSuppliers([]);
          setLoadError(
            err instanceof PublicSupplierApiError
              ? err.message
              : '取引先一覧の取得に失敗しました',
          );
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    load();
    return () => {
      cancelled = true;
    };
  }, []);

  if (isLoading) {
    return (
      <div className="space-y-6">
        {[1, 2].map((key) => (
          <div key={key} className="animate-pulse overflow-hidden rounded-xl border border-gray-200 bg-white">
            <div className="h-48 bg-gray-100" />
            <div className="space-y-3 p-5">
              <div className="h-5 w-1/3 rounded bg-gray-100" />
              <div className="h-4 w-full rounded bg-gray-100" />
              <div className="h-4 w-2/3 rounded bg-gray-100" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (loadError) {
    return <p className="py-8 text-center text-sm text-red-500">{loadError}</p>;
  }

  if (suppliers.length === 0) {
    return <p className="py-12 text-center text-gray-400">現在、公開中の取引先はありません</p>;
  }

  return (
    <div className="space-y-6">
      {suppliers.map((supplier) => (
        <SupplierCard key={supplier.id} supplier={supplier} />
      ))}
    </div>
  );
}
