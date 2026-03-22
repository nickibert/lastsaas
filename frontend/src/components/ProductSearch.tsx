import { useState, useRef, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { productsApi, type Product } from '../api/procurement';
import { Search, Loader2 } from 'lucide-react';

interface Props {
  value?: string;          // current productId
  currentName?: string;    // display name when not searching
  onChange: (id: string, product: Product) => void;
  supplierId?: string;     // if set, prefer products from this supplier
  className?: string;
}

export function ProductSearch({ value, currentName, onChange, supplierId, className = '' }: Props) {
  const [open, setOpen] = useState(false);
  const [q, setQ] = useState('');
  const [ignoreSupplier, setIgnoreSupplier] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // When supplier filter is active, pass it through — unless user toggled it off
  const effectiveSupplierId = (supplierId && !ignoreSupplier) ? supplierId : undefined;

  const { data, isLoading } = useQuery({
    queryKey: ['product-search', q, effectiveSupplierId],
    queryFn: () => productsApi.list(q || undefined, 1, 30, effectiveSupplierId),
    enabled: open,
    throwOnError: false,
    staleTime: 15_000,
  });
  const results = data?.items ?? [];

  // Reset ignoreSupplier when supplier changes
  useEffect(() => { setIgnoreSupplier(false); }, [supplierId]);

  // Close on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
        setQ('');
        setIgnoreSupplier(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const displayName = currentName ?? (value ? `ID: ${value}` : '— auswählen —');

  const handleSelect = (p: Product) => {
    onChange(p.id, p);
    setOpen(false);
    setQ('');
    setIgnoreSupplier(false);
  };

  return (
    <div ref={containerRef} className={`relative ${className}`}>
      {open ? (
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-dark-500 pointer-events-none" />
          <input
            ref={inputRef}
            autoFocus
            type="text"
            value={q}
            onChange={e => setQ(e.target.value)}
            placeholder="Produkt suchen…"
            className="w-full pl-7 pr-7 py-1 bg-dark-800 border border-primary-500 rounded text-white text-sm focus:outline-none"
          />
          {isLoading && (
            <Loader2 className="absolute right-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-dark-500 animate-spin" />
          )}
        </div>
      ) : (
        <button
          type="button"
          onClick={() => { setOpen(true); setQ(''); }}
          className="w-full text-left px-2 py-1 bg-dark-800 border border-dark-700 rounded text-sm truncate hover:border-dark-500 focus:outline-none focus:border-primary-500"
          style={{ color: value ? 'white' : '#6b7280' }}
        >
          {displayName}
        </button>
      )}

      {open && (
        <div className="absolute z-50 top-full left-0 mt-1 w-80 bg-dark-900 border border-dark-700 rounded-lg shadow-xl overflow-hidden">
          {/* Supplier filter indicator */}
          {supplierId && (
            <div className="flex items-center justify-between px-3 py-1.5 bg-dark-800/60 border-b border-dark-700">
              <span className="text-xs text-dark-500">
                {ignoreSupplier ? 'Alle Produkte' : 'Nur Lieferanten-Produkte'}
              </span>
              <button
                type="button"
                onMouseDown={e => e.preventDefault()}
                onClick={() => setIgnoreSupplier(v => !v)}
                className="text-xs text-primary-400 hover:text-primary-300 transition-colors"
              >
                {ignoreSupplier ? 'Lieferant filtern' : 'Alle anzeigen'}
              </button>
            </div>
          )}

          <div className="max-h-64 overflow-y-auto">
            {isLoading ? (
              <div className="flex items-center gap-2 px-3 py-4 text-sm text-dark-400">
                <Loader2 className="w-4 h-4 animate-spin flex-shrink-0" />
                <span>Suche läuft…</span>
              </div>
            ) : results.length === 0 ? (
              <div>
                <div className="px-3 py-3 text-sm text-dark-400">
                  {q
                    ? `Kein Treffer für „${q}"`
                    : supplierId && !ignoreSupplier
                      ? 'Keine Produkte für diesen Lieferanten'
                      : 'Keine Produkte gefunden'}
                </div>
                {/* Offer to remove supplier filter if that might be the issue */}
                {supplierId && !ignoreSupplier && (
                  <button
                    type="button"
                    onMouseDown={e => e.preventDefault()}
                    onClick={() => setIgnoreSupplier(true)}
                    className="w-full text-left px-3 py-2 text-sm text-primary-400 hover:bg-dark-800 border-t border-dark-800 transition-colors"
                  >
                    Alle Produkte anzeigen →
                  </button>
                )}
              </div>
            ) : (
              results.map(p => (
                <button
                  key={p.id}
                  type="button"
                  onMouseDown={e => e.preventDefault()}
                  onClick={() => handleSelect(p)}
                  className="w-full text-left px-3 py-2 text-sm hover:bg-dark-700 transition-colors"
                >
                  <div className="text-white font-mono leading-tight">{p.nameShort}</div>
                  {p.ownNameShort && p.ownNameShort !== p.nameShort && (
                    <div className="text-dark-400 text-xs mt-0.5">{p.ownNameShort}</div>
                  )}
                  {p.nameLong && p.nameLong !== p.nameShort && (
                    <div className="text-dark-500 text-xs mt-0.5 truncate">{p.nameLong}</div>
                  )}
                </button>
              ))
            )}
          </div>

          {/* Result count when there are results */}
          {!isLoading && results.length > 0 && (
            <div className="border-t border-dark-800 px-3 py-1.5 text-xs text-dark-600">
              {results.length === 30 ? '30+ Treffer – Suche verfeinern' : `${results.length} Treffer`}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
