import { useState, useRef, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { productsApi, type Product } from '../api/procurement';

interface Props {
  value?: string;          // current productId
  currentName?: string;    // display name when not searching
  onChange: (id: string, product: Product) => void;
  className?: string;
}

export function ProductSearch({ value, currentName, onChange, className = '' }: Props) {
  const [open, setOpen] = useState(false);
  const [q, setQ] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const { data } = useQuery({
    queryKey: ['product-search', q],
    queryFn: () => productsApi.list(q || undefined, 1),
    enabled: open,
    throwOnError: false,
  });
  const results = data?.items ?? [];

  // Close on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
        setQ('');
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const displayName = currentName ?? (value ? `ID: ${value}` : '— auswählen —');

  return (
    <div ref={containerRef} className={`relative ${className}`}>
      {open ? (
        <input
          ref={inputRef}
          autoFocus
          type="text"
          value={q}
          onChange={e => setQ(e.target.value)}
          placeholder="Produkt suchen..."
          className="w-full px-2 py-1 bg-dark-800 border border-primary-500 rounded text-white text-sm focus:outline-none"
        />
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
        <div className="absolute z-50 top-full left-0 mt-1 w-72 bg-dark-900 border border-dark-700 rounded-lg shadow-xl max-h-60 overflow-y-auto">
          {results.length === 0 ? (
            <div className="px-3 py-2 text-sm text-dark-400">
              {q ? 'Kein Treffer' : 'Suchbegriff eingeben...'}
            </div>
          ) : (
            results.map(p => (
              <button
                key={p.id}
                type="button"
                onMouseDown={e => e.preventDefault()}
                onClick={() => { onChange(p.id, p); setOpen(false); setQ(''); }}
                className="w-full text-left px-3 py-2 text-sm hover:bg-dark-700 text-white"
              >
                <span className="font-mono">{p.nameShort}</span>
                {p.ownNameShort && p.ownNameShort !== p.nameShort && (
                  <span className="text-dark-400 ml-2">({p.ownNameShort})</span>
                )}
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}
