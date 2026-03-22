import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Search, X, LayoutDashboard, Users, Settings, CreditCard, ArrowRight,
  CalendarDays, Package, Tag, Warehouse, Box, Globe, Anchor, Ship,
  ShoppingCart, UserCheck, BarChart3, Layers, Building2,
} from 'lucide-react';

interface CommandItem {
  id: string;
  label: string;
  sublabel?: string;
  icon: React.ElementType;
  path: string;
  keywords?: string[];
}

const COMMANDS: CommandItem[] = [
  { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard, path: '/dashboard', keywords: ['start', 'home'] },
  { id: 'proc-overview', label: 'Übersicht', sublabel: 'Einkauf', icon: BarChart3, path: '/procurement', keywords: ['einkauf', 'übersicht'] },
  { id: 'calendar', label: 'Kalender', sublabel: 'Einkauf', icon: CalendarDays, path: '/procurement/calendar', keywords: ['kalender', 'lieferungen', 'termine'] },
  { id: 'orders', label: 'Bestellungen', sublabel: 'Einkauf', icon: ShoppingCart, path: '/procurement/orders', keywords: ['bestellungen', 'orders'] },
  { id: 'offers', label: 'Angebote', sublabel: 'Einkauf', icon: Tag, path: '/procurement/offers', keywords: ['angebote', 'offers'] },
  { id: 'inventory', label: 'Lager', sublabel: 'Einkauf', icon: Warehouse, path: '/procurement/inventory', keywords: ['lager', 'inventar', 'bestand'] },
  { id: 'products', label: 'Produkte', sublabel: 'Stammdaten', icon: Package, path: '/procurement/products', keywords: ['produkte', 'artikel'] },
  { id: 'goods-groups', label: 'Warengruppen', sublabel: 'Stammdaten', icon: Layers, path: '/procurement/goods-groups', keywords: ['warengruppen', 'kategorien'] },
  { id: 'suppliers', label: 'Lieferanten', sublabel: 'Stammdaten', icon: Building2, path: '/procurement/suppliers', keywords: ['lieferanten', 'supplier'] },
  { id: 'customers', label: 'Kunden', sublabel: 'Stammdaten', icon: UserCheck, path: '/procurement/customers', keywords: ['kunden', 'customers'] },
  { id: 'harbours', label: 'Häfen', sublabel: 'Logistik', icon: Anchor, path: '/procurement/harbours', keywords: ['häfen', 'hafen', 'ports'] },
  { id: 'containers', label: 'Container', sublabel: 'Logistik', icon: Box, path: '/procurement/containers', keywords: ['container'] },
  { id: 'freight', label: 'Frachtführer', sublabel: 'Logistik', icon: Ship, path: '/procurement/freight-carriers', keywords: ['frachtführer', 'spedition', 'transport'] },
  { id: 'countries', label: 'Länder', sublabel: 'Logistik', icon: Globe, path: '/procurement/countries', keywords: ['länder', 'countries'] },
  { id: 'team', label: 'Team', icon: Users, path: '/team', keywords: ['team', 'mitglieder'] },
  { id: 'plan', label: 'Plan & Abonnement', icon: CreditCard, path: '/plan', keywords: ['plan', 'billing', 'abonnement'] },
  { id: 'settings', label: 'Einstellungen', icon: Settings, path: '/settings', keywords: ['einstellungen', 'settings', 'profil'] },
];

interface Props {
  open: boolean;
  onClose: () => void;
}

export default function CommandPalette({ open, onClose }: Props) {
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const [selectedIdx, setSelectedIdx] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const filtered = query.trim()
    ? COMMANDS.filter(c => {
        const q = query.toLowerCase();
        return (
          c.label.toLowerCase().includes(q) ||
          c.sublabel?.toLowerCase().includes(q) ||
          c.keywords?.some(k => k.includes(q))
        );
      })
    : COMMANDS;

  useEffect(() => {
    if (open) {
      setQuery('');
      setSelectedIdx(0);
      setTimeout(() => inputRef.current?.focus(), 10);
    }
  }, [open]);

  useEffect(() => { setSelectedIdx(0); }, [query]);

  const select = useCallback((item: CommandItem) => {
    navigate(item.path);
    onClose();
  }, [navigate, onClose]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (!open) return;
      if (e.key === 'Escape') { onClose(); return; }
      if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIdx(i => Math.min(i + 1, filtered.length - 1)); }
      if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedIdx(i => Math.max(i - 1, 0)); }
      if (e.key === 'Enter' && filtered[selectedIdx]) { select(filtered[selectedIdx]); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, filtered, selectedIdx, select, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh]" onClick={onClose}>
      <div className="absolute inset-0 bg-dark-950/70 backdrop-blur-sm" />
      <div
        className="relative w-full max-w-xl mx-4 bg-dark-900 border border-dark-700 rounded-2xl shadow-2xl overflow-hidden"
        onClick={e => e.stopPropagation()}
      >
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 py-3.5 border-b border-dark-800">
          <Search className="w-5 h-5 text-dark-400 flex-shrink-0" />
          <input
            ref={inputRef}
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Seite suchen oder navigieren…"
            className="flex-1 bg-transparent text-dark-100 placeholder-dark-500 outline-none text-sm"
          />
          <button onClick={onClose} className="text-dark-500 hover:text-dark-300 transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Results */}
        <div className="max-h-80 overflow-y-auto py-1.5">
          {filtered.length === 0 ? (
            <div className="px-4 py-10 text-center text-dark-500 text-sm">Keine Ergebnisse für „{query}"</div>
          ) : (
            filtered.map((item, idx) => (
              <button
                key={item.id}
                onClick={() => select(item)}
                onMouseEnter={() => setSelectedIdx(idx)}
                className={`w-full flex items-center gap-3 px-4 py-2.5 text-left transition-colors ${
                  idx === selectedIdx
                    ? 'bg-primary-500/12 text-primary-400'
                    : 'text-dark-300 hover:bg-dark-800/60'
                }`}
              >
                <item.icon className="w-4 h-4 flex-shrink-0 opacity-70" />
                <div className="flex-1 min-w-0">
                  <span className="text-sm font-medium">{item.label}</span>
                  {item.sublabel && (
                    <span className="text-xs text-dark-500 ml-2">{item.sublabel}</span>
                  )}
                </div>
                {idx === selectedIdx && <ArrowRight className="w-3.5 h-3.5 flex-shrink-0 opacity-60" />}
              </button>
            ))
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-dark-800/60 px-4 py-2 flex items-center gap-4 text-xs text-dark-600">
          <span><kbd className="font-mono bg-dark-800 px-1.5 py-0.5 rounded text-dark-400">↑↓</kbd> navigieren</span>
          <span><kbd className="font-mono bg-dark-800 px-1.5 py-0.5 rounded text-dark-400">↵</kbd> öffnen</span>
          <span><kbd className="font-mono bg-dark-800 px-1.5 py-0.5 rounded text-dark-400">Esc</kbd> schließen</span>
        </div>
      </div>
    </div>
  );
}
