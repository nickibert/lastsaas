import { useState, useRef, useEffect, useCallback } from 'react';
import { createPortal } from 'react-dom';
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
  // Position of the dropdown in viewport coordinates (for portal rendering)
  const [dropdownPos, setDropdownPos] = useState({ top: 0, left: 0, width: 0 });

  const triggerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

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

  // Calculate dropdown position from trigger element
  const updatePosition = useCallback(() => {
    if (!triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    // Check if there's room below; if not, open upward
    const spaceBelow = window.innerHeight - rect.bottom;
    const dropdownHeight = 320; // approximate max height
    const top = spaceBelow >= dropdownHeight
      ? rect.bottom + 4
      : rect.top - dropdownHeight - 4;
    setDropdownPos({ top, left: rect.left, width: Math.max(rect.width, 320) });
  }, []);

  // Open: calculate position and focus input
  const handleOpen = () => {
    updatePosition();
    setOpen(true);
    setQ('');
  };

  // Reposition on scroll/resize while open
  useEffect(() => {
    if (!open) return;
    const reposition = () => updatePosition();
    window.addEventListener('scroll', reposition, true);
    window.addEventListener('resize', reposition);
    return () => {
      window.removeEventListener('scroll', reposition, true);
      window.removeEventListener('resize', reposition);
    };
  }, [open, updatePosition]);

  // Focus input after open
  useEffect(() => {
    if (open) setTimeout(() => inputRef.current?.focus(), 10);
  }, [open]);

  // Close on outside click (checks both trigger and portal dropdown)
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      const insideTrigger = triggerRef.current?.contains(target);
      const insideDropdown = dropdownRef.current?.contains(target);
      if (!insideTrigger && !insideDropdown) {
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

  const dropdown = open ? (
    <div
      ref={dropdownRef}
      style={{ position: 'fixed', top: dropdownPos.top, left: dropdownPos.left, width: dropdownPos.width, zIndex: 9999 }}
      className="bg-dark-900 border border-dark-700 rounded-lg shadow-2xl overflow-hidden"
    >
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

      {!isLoading && results.length > 0 && (
        <div className="border-t border-dark-800 px-3 py-1.5 text-xs text-dark-600">
          {results.length === 30 ? '30+ Treffer – Suche verfeinern' : `${results.length} Treffer`}
        </div>
      )}
    </div>
  ) : null;

  return (
    <div ref={triggerRef} className={`relative ${className}`}>
      {open ? (
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-dark-500 pointer-events-none" />
          <input
            ref={inputRef}
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
          onClick={handleOpen}
          className="w-full text-left px-2 py-1 bg-dark-800 border border-dark-700 rounded text-sm truncate hover:border-dark-500 focus:outline-none focus:border-primary-500"
          style={{ color: value ? 'white' : '#6b7280' }}
        >
          {displayName}
        </button>
      )}

      {/* Render dropdown via portal so overflow:hidden on table parents can't clip it */}
      {createPortal(dropdown, document.body)}
    </div>
  );
}
