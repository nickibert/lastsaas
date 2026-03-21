import { ChevronLeft, ChevronRight } from 'lucide-react';

interface Props {
  page: number;
  pages: number;
  total: number;
  limit: number;
  onPage: (p: number) => void;
  onLimit: (l: number) => void;
}

const PAGE_SIZES = [10, 25, 50, 100];

export function Pagination({ page, pages, total, limit, onPage, onLimit }: Props) {
  if (total === 0) return null;
  const start = Math.max(1, Math.min(page - 3, pages - 6));
  const pageNums = Array.from({ length: Math.min(7, pages) }, (_, i) => start + i);

  return (
    <div className="flex items-center justify-between px-4 py-3 border-t border-dark-800">
      <div className="flex items-center gap-3">
        <span className="text-sm text-dark-400">
          {total.toLocaleString('de-DE')} Einträge
        </span>
        <select
          value={limit}
          onChange={e => { onLimit(Number(e.target.value)); onPage(1); }}
          className="px-2 py-1 bg-dark-800 border border-dark-700 rounded text-sm text-white focus:outline-none focus:border-primary-500"
        >
          {PAGE_SIZES.map(s => <option key={s} value={s}>{s} pro Seite</option>)}
        </select>
      </div>
      {pages > 1 && (
        <div className="flex items-center gap-1">
          <button onClick={() => onPage(Math.max(1, page - 1))} disabled={page === 1}
            className="p-1.5 rounded text-dark-400 hover:text-white hover:bg-dark-700 disabled:opacity-30 disabled:cursor-not-allowed">
            <ChevronLeft className="w-4 h-4" />
          </button>
          {pageNums.map(p => (
            <button key={p} onClick={() => onPage(p)}
              className={`w-8 h-8 rounded text-sm transition-colors ${p === page ? 'bg-primary-500 text-white' : 'text-dark-400 hover:text-white hover:bg-dark-700'}`}>
              {p}
            </button>
          ))}
          <button onClick={() => onPage(Math.min(pages, page + 1))} disabled={page === pages}
            className="p-1.5 rounded text-dark-400 hover:text-white hover:bg-dark-700 disabled:opacity-30 disabled:cursor-not-allowed">
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      )}
    </div>
  );
}
