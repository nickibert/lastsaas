import { LayoutList } from 'lucide-react';
import { useLocalStorage } from '../../../hooks/useLocalStorage';

const PAGE_SIZES = [10, 25, 50, 100];

export default function AppearanceTab() {
  const [pageSize, setPageSize] = useLocalStorage<number>('table_page_size', 25);

  return (
    <div className="space-y-6 max-w-2xl">
      <div className="bg-dark-900/50 backdrop-blur-sm border border-dark-800 rounded-2xl p-6">
        <h2 className="text-lg font-semibold text-white flex items-center gap-2 mb-4">
          <LayoutList className="w-5 h-5 text-dark-400" />
          Tabellenansicht
        </h2>
        <div className="space-y-4">
          <div className="flex items-center justify-between py-2">
            <div>
              <p className="text-sm text-white">Einträge pro Seite</p>
              <p className="text-xs text-dark-400 mt-0.5">Standard-Seitengröße für alle Listen</p>
            </div>
            <select
              value={pageSize}
              onChange={e => setPageSize(Number(e.target.value))}
              className="px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-sm text-white focus:outline-none focus:border-primary-500"
            >
              {PAGE_SIZES.map(s => (
                <option key={s} value={s}>{s} Einträge</option>
              ))}
            </select>
          </div>
        </div>
      </div>
    </div>
  );
}
