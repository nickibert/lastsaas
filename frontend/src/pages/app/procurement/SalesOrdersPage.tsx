import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronDown, ChevronUp, Search } from 'lucide-react';
import { salesOrdersApi } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';
import { Pagination } from '../../../components/Pagination';
import { useLocalStorage } from '../../../hooks/useLocalStorage';

const statusBadge: Record<string, string> = {
  open:      'bg-indigo-500/20 text-indigo-400',
  confirmed: 'bg-blue-500/20 text-blue-400',
  shipped:   'bg-teal-500/20 text-teal-400',
  delivered: 'bg-emerald-500/20 text-emerald-400',
  cancelled: 'bg-red-500/20 text-red-400',
};

function fmt(n?: number) {
  if (n == null) return '–';
  return n.toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function fmtDate(s?: string) {
  if (!s) return '–';
  return new Date(s).toLocaleDateString('de-DE');
}

export default function SalesOrdersPage() {
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useLocalStorage<number>('table_page_size', 25);
  const [expanded, setExpanded] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['sales-orders', search, status, page, limit],
    queryFn: () => salesOrdersApi.list(search || undefined, status || undefined, page, limit),
    enabled,
    throwOnError: false,
    placeholderData: prev => prev,
  });

  const orders = data?.items ?? [];
  const total = data?.total ?? 0;
  const pages = data?.pages ?? 1;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-white">Verkaufsaufträge</h1>
          <p className="text-sm text-dark-400 mt-0.5">Aus Xentral synchronisierte Aufträge</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-dark-400 pointer-events-none" />
          <input
            type="text"
            value={search}
            onChange={e => { setSearch(e.target.value); setPage(1); }}
            placeholder="Suche nach Belegnummer…"
            className="w-full pl-9 pr-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500"
          />
        </div>
        <select
          value={status}
          onChange={e => { setStatus(e.target.value); setPage(1); }}
          className="px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500"
        >
          <option value="">Alle Status</option>
          <option value="open">Offen</option>
          <option value="confirmed">Bestätigt</option>
          <option value="shipped">Versendet</option>
          <option value="delivered">Geliefert</option>
          <option value="cancelled">Storniert</option>
        </select>
      </div>

      {/* Table */}
      <div className="bg-dark-900 rounded-xl border border-dark-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-dark-800">
              <th className="px-4 py-3 text-left text-dark-400 font-medium w-8"></th>
              <th className="px-4 py-3 text-left text-dark-400 font-medium">Belegnummer</th>
              <th className="px-4 py-3 text-left text-dark-400 font-medium">Ext. Auftragnr.</th>
              <th className="px-4 py-3 text-left text-dark-400 font-medium">Datum</th>
              <th className="px-4 py-3 text-left text-dark-400 font-medium">Status</th>
              <th className="px-4 py-3 text-right text-dark-400 font-medium">Positionen</th>
              <th className="px-4 py-3 text-right text-dark-400 font-medium">Netto (€)</th>
              <th className="px-4 py-3 text-right text-dark-400 font-medium">Brutto (€)</th>
            </tr>
          </thead>
          <tbody>
            {isLoading && (
              <tr><td colSpan={8} className="px-4 py-8 text-center text-dark-400">Lade…</td></tr>
            )}
            {!isLoading && orders.length === 0 && (
              <tr><td colSpan={8} className="px-4 py-8 text-center text-dark-400">Keine Verkaufsaufträge gefunden</td></tr>
            )}
            {orders.map(so => (
              <>
                <tr
                  key={so.id}
                  className="border-b border-dark-800 hover:bg-dark-800/50 cursor-pointer"
                  onClick={() => setExpanded(expanded === so.id ? null : so.id)}
                >
                  <td className="px-4 py-3 text-dark-400">
                    {(so.lineItems?.length ?? 0) > 0
                      ? expanded === so.id
                        ? <ChevronUp className="w-4 h-4" />
                        : <ChevronDown className="w-4 h-4" />
                      : null}
                  </td>
                  <td className="px-4 py-3 text-white font-mono text-xs">{so.xentralDocumentNr || '–'}</td>
                  <td className="px-4 py-3 text-dark-300 text-xs">{so.externalOrderNr || '–'}</td>
                  <td className="px-4 py-3 text-dark-300">{fmtDate(so.date)}</td>
                  <td className="px-4 py-3">
                    {so.status ? (
                      <span className={`inline-flex px-2 py-0.5 rounded-full text-xs font-medium ${statusBadge[so.status] ?? 'bg-dark-700 text-dark-300'}`}>
                        {so.status}
                      </span>
                    ) : '–'}
                  </td>
                  <td className="px-4 py-3 text-right text-dark-300">{so.lineItems?.length ?? 0}</td>
                  <td className="px-4 py-3 text-right text-dark-300 font-mono">{fmt(so.totalNetEur)}</td>
                  <td className="px-4 py-3 text-right text-dark-300 font-mono">{fmt(so.totalGrossEur)}</td>
                </tr>
                {expanded === so.id && (so.lineItems?.length ?? 0) > 0 && (
                  <tr key={so.id + '-detail'} className="border-b border-dark-800 bg-dark-900/60">
                    <td colSpan={8} className="px-8 py-3">
                      <table className="w-full text-xs">
                        <thead>
                          <tr className="text-dark-400">
                            <th className="text-left py-1 pr-4">Artikel-ID</th>
                            <th className="text-left py-1 pr-4">Beschreibung</th>
                            <th className="text-right py-1 pr-4">Menge</th>
                            <th className="text-right py-1 pr-4">EP (€)</th>
                            <th className="text-right py-1">Gesamt (€)</th>
                          </tr>
                        </thead>
                        <tbody>
                          {so.lineItems!.map((li, i) => (
                            <tr key={i} className="border-t border-dark-800">
                              <td className="py-1 pr-4 font-mono text-dark-400">{li.xentralArticleId || '–'}</td>
                              <td className="py-1 pr-4 text-dark-300">{li.description || '–'}</td>
                              <td className="py-1 pr-4 text-right text-dark-300">{li.quantity}</td>
                              <td className="py-1 pr-4 text-right font-mono text-dark-300">{fmt(li.unitPriceEur)}</td>
                              <td className="py-1 text-right font-mono text-dark-300">{fmt(li.totalPriceEur)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </td>
                  </tr>
                )}
              </>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pages={pages} total={total} limit={limit} onPage={setPage} onLimit={setLimit} />
    </div>
  );
}
