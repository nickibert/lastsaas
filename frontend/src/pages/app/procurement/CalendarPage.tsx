import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight, CheckSquare, Ship, Plus, Pencil, Trash2, X, ExternalLink } from 'lucide-react';
import { Link } from 'react-router-dom';
import { toast } from 'sonner';
import {
  calendarApi, calendarEntriesApi, ordersApi,
  type CalendarTask, type CalendarArrival, type CalendarEntry, type CalendarEntryType,
} from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function fmt(date: Date) {
  return date.toISOString().slice(0, 10);
}

function daysInMonth(year: number, month: number) {
  return new Date(year, month + 1, 0).getDate();
}

function firstDayOfMonth(year: number, month: number) {
  const d = new Date(year, month, 1).getDay();
  return (d + 6) % 7; // Mon=0 … Sun=6
}

const MONTHS_DE = ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'];
const DAYS_DE = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So'];

const ENTRY_TYPE_LABELS: Record<CalendarEntryType, string> = {
  reminder: 'Erinnerung',
  milestone: 'Meilenstein',
  appointment: 'Termin',
  deadline: 'Deadline',
};

const ENTRY_TYPE_COLORS: Record<CalendarEntryType, string> = {
  reminder:    'bg-yellow-500/20 text-yellow-300',
  milestone:   'bg-purple-500/20 text-purple-300',
  appointment: 'bg-sky-500/20 text-sky-300',
  deadline:    'bg-red-500/20 text-red-300',
};

// ---------------------------------------------------------------------------
// Entry form modal
// ---------------------------------------------------------------------------

interface EntryFormProps {
  initial?: Partial<CalendarEntry>;
  defaultDate?: string;
  onSave: (data: Partial<CalendarEntry>) => void;
  onClose: () => void;
  saving: boolean;
}

function EntryForm({ initial, defaultDate, onSave, onClose, saving }: EntryFormProps) {
  const [form, setForm] = useState<Partial<CalendarEntry>>({
    title: '',
    notes: '',
    entryType: 'reminder',
    date: defaultDate ?? fmt(new Date()),
    ...initial,
  });

  const inp = (key: keyof CalendarEntry) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
    setForm(f => ({ ...f, [key]: e.target.value }));

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-dark-900 border border-dark-700 rounded-2xl p-6 w-full max-w-md space-y-4 shadow-2xl">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-white">
            {initial?.id ? 'Eintrag bearbeiten' : 'Neuer Kalendereintrag'}
          </h2>
          <button onClick={onClose} className="p-1 text-dark-400 hover:text-white">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div>
          <label className="block text-xs text-dark-400 mb-1">Titel *</label>
          <input type="text" value={form.title ?? ''} onChange={inp('title')} autoFocus
            className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500" />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-dark-400 mb-1">Datum *</label>
            <input type="date" value={form.date?.slice(0, 10) ?? ''} onChange={inp('date')}
              className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500" />
          </div>
          <div>
            <label className="block text-xs text-dark-400 mb-1">Typ</label>
            <select value={form.entryType ?? 'reminder'} onChange={inp('entryType')}
              className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500">
              {(Object.keys(ENTRY_TYPE_LABELS) as CalendarEntryType[]).map(t => (
                <option key={t} value={t}>{ENTRY_TYPE_LABELS[t]}</option>
              ))}
            </select>
          </div>
        </div>

        <div>
          <label className="block text-xs text-dark-400 mb-1">Notiz</label>
          <textarea value={form.notes ?? ''} onChange={inp('notes')} rows={3}
            className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500 resize-none" />
        </div>

        <div className="flex gap-3 pt-1">
          <button onClick={() => onSave(form)} disabled={!form.title || !form.date || saving}
            className="flex-1 py-2 bg-primary-500 text-white rounded-lg text-sm hover:bg-primary-600 disabled:opacity-50">
            {saving ? 'Speichert…' : 'Speichern'}
          </button>
          <button onClick={onClose} className="flex-1 py-2 bg-dark-700 text-white rounded-lg text-sm hover:bg-dark-600">
            Abbrechen
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Freight date modal
// ---------------------------------------------------------------------------

type FreightDateKey = 'shippingDate' | 'estimatedArrival' | 'arrival' | 'avisShipperDate';

const FREIGHT_DATE_FIELDS: { key: FreightDateKey; label: string }[] = [
  { key: 'shippingDate',     label: 'Versanddatum' },
  { key: 'estimatedArrival', label: 'Voraussichtliche Ankunft (ETA)' },
  { key: 'arrival',          label: 'Tatsächliche Ankunft' },
  { key: 'avisShipperDate',  label: 'Avis Spediteur' },
];

function defaultDateForField(field: FreightDateKey, arrival: CalendarArrival): string {
  if (field === 'estimatedArrival') return arrival.estimatedDate ?? '';
  if (field === 'arrival')          return arrival.actualDate ?? '';
  return '';
}

interface FreightDateModalProps {
  arrival: CalendarArrival;
  onClose: () => void;
}

function FreightDateModal({ arrival, onClose }: FreightDateModalProps) {
  const qc = useQueryClient();
  const [field, setField] = useState<FreightDateKey>('estimatedArrival');
  const [date, setDate] = useState(arrival.estimatedDate ?? '');

  const handleFieldChange = (f: FreightDateKey) => {
    setField(f);
    setDate(defaultDateForField(f, arrival));
  };

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ['calendar'] });
    qc.invalidateQueries({ queryKey: ['order', arrival.orderId] });
  };

  const mut = useMutation({
    mutationFn: () => ordersApi.patchFreightDate(arrival.orderId, arrival.freightIndex, field, date),
    onSuccess: () => { invalidate(); toast.success('Datum gespeichert'); onClose(); },
    onError: () => toast.error('Fehler beim Speichern'),
  });

  const clearMut = useMutation({
    mutationFn: () => ordersApi.patchFreightDate(arrival.orderId, arrival.freightIndex, field, ''),
    onSuccess: () => { invalidate(); toast.success('Datum gelöscht'); onClose(); },
    onError: () => toast.error('Fehler beim Löschen'),
  });

  const currentDate = defaultDateForField(field, arrival);
  const isPending = mut.isPending || clearMut.isPending;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-dark-900 border border-dark-700 rounded-2xl p-6 w-full max-w-md space-y-4 shadow-2xl">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-white">Frachtdatum bearbeiten</h2>
          <button onClick={onClose} className="p-1 text-dark-400 hover:text-white">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="text-sm text-dark-400">
          <span className="text-white font-medium">{arrival.orderNumber}</span>
          {arrival.supplierName && <span> – {arrival.supplierName}</span>}
        </div>

        <div>
          <label className="block text-xs text-dark-400 mb-1">Datumsfeld</label>
          <select value={field} onChange={e => handleFieldChange(e.target.value as FreightDateKey)}
            className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500">
            {FREIGHT_DATE_FIELDS.map(f => (
              <option key={f.key} value={f.key}>{f.label}</option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-xs text-dark-400 mb-1">Datum</label>
          <input type="date" value={date} onChange={e => setDate(e.target.value)} autoFocus
            className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500" />
        </div>

        <div className="flex gap-3 pt-1">
          <button onClick={() => mut.mutate()} disabled={!date || isPending}
            className="flex-1 py-2 bg-primary-500 text-white rounded-lg text-sm hover:bg-primary-600 disabled:opacity-50">
            {mut.isPending ? 'Speichert…' : 'Speichern'}
          </button>
          {currentDate && (
            <button onClick={() => clearMut.mutate()} disabled={isPending}
              className="px-4 py-2 bg-red-500/20 text-red-400 border border-red-500/30 rounded-lg text-sm hover:bg-red-500/30 disabled:opacity-50"
              title="Datum löschen">
              <Trash2 className="w-4 h-4" />
            </button>
          )}
          <button onClick={onClose} className="flex-1 py-2 bg-dark-700 text-white rounded-lg text-sm hover:bg-dark-600">
            Abbrechen
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Day events types
// ---------------------------------------------------------------------------

interface DayEvents {
  tasks: CalendarTask[];
  arrivals: CalendarArrival[];
  entries: CalendarEntry[];
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function CalendarPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;

  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth());

  // Entry form state
  const [showForm, setShowForm] = useState(false);
  const [editingEntry, setEditingEntry] = useState<CalendarEntry | null>(null);
  const [formDefaultDate, setFormDefaultDate] = useState<string | undefined>();

  // Freight date modal state
  const [freightModalArrival, setFreightModalArrival] = useState<CalendarArrival | null>(null);

  const fromDate = new Date(year, month, 1);
  const toDate   = new Date(year, month + 1, 0);

  const { data, isLoading } = useQuery({
    queryKey: ['calendar', year, month],
    queryFn: () => calendarApi.get(fmt(fromDate), fmt(toDate)),
    enabled,
    throwOnError: false,
  });

  const tasks    = data?.tasks    ?? [];
  const arrivals = data?.arrivals ?? [];
  const entries  = data?.entries  ?? [];

  const createMut = useMutation({
    mutationFn: calendarEntriesApi.create,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['calendar'] });
      setShowForm(false);
      toast.success('Eintrag angelegt');
    },
    onError: () => toast.error('Fehler beim Anlegen'),
  });

  const updateMut = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CalendarEntry> }) =>
      calendarEntriesApi.update(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['calendar'] });
      setEditingEntry(null);
      toast.success('Gespeichert');
    },
    onError: () => toast.error('Fehler'),
  });

  const deleteMut = useMutation({
    mutationFn: calendarEntriesApi.delete,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['calendar'] });
      toast.success('Gelöscht');
    },
    onError: () => toast.error('Fehler'),
  });

  const handleSave = (form: Partial<CalendarEntry>) => {
    // Convert "YYYY-MM-DD" to RFC3339 for Go
    const payload = { ...form };
    if (payload.date && !payload.date.includes('T')) {
      payload.date = payload.date + 'T00:00:00Z';
    }
    if (editingEntry) {
      updateMut.mutate({ id: editingEntry.id, data: payload });
    } else {
      createMut.mutate(payload);
    }
  };

  const openNewEntry = (dateStr?: string) => {
    setEditingEntry(null);
    setFormDefaultDate(dateStr);
    setShowForm(true);
  };

  // Build event map
  const eventMap: Record<string, DayEvents> = {};
  const ensure = (d: string) => { if (!eventMap[d]) eventMap[d] = { tasks: [], arrivals: [], entries: [] }; };

  tasks.forEach(t => { ensure(t.dueDate); eventMap[t.dueDate].tasks.push(t); });
  arrivals.forEach(a => {
    const d = a.actualDate || a.estimatedDate;
    if (d) { ensure(d); eventMap[d].arrivals.push(a); }
  });
  entries.forEach(e => {
    const d = e.date.slice(0, 10);
    ensure(d);
    eventMap[d].entries.push(e);
  });

  const prevMonth = () => { if (month === 0) { setMonth(11); setYear(y => y - 1); } else setMonth(m => m - 1); };
  const nextMonth = () => { if (month === 11) { setMonth(0); setYear(y => y + 1); } else setMonth(m => m + 1); };

  const days     = daysInMonth(year, month);
  const firstDay = firstDayOfMonth(year, month);
  const todayStr = fmt(now);

  const cells: (number | null)[] = [];
  for (let i = 0; i < firstDay; i++) cells.push(null);
  for (let d = 1; d <= days; d++) cells.push(d);
  while (cells.length % 7 !== 0) cells.push(null);

  const openTasks = tasks.filter(t => !t.doneAt).length;
  const saving = createMut.isPending || updateMut.isPending;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Kalender</h1>
          <p className="text-dark-400 mt-1">
            {tasks.length} Aufgaben ({openTasks} offen) · {arrivals.length} Ankünfte · {entries.length} Einträge
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => openNewEntry()}
            className="flex items-center gap-1.5 px-3 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 text-sm">
            <Plus className="w-4 h-4" /> Termin
          </button>
          <button onClick={prevMonth} className="p-2 rounded-lg bg-dark-800 text-dark-300 hover:text-white hover:bg-dark-700">
            <ChevronLeft className="w-5 h-5" />
          </button>
          <span className="text-white font-semibold min-w-[160px] text-center">
            {MONTHS_DE[month]} {year}
          </span>
          <button onClick={nextMonth} className="p-2 rounded-lg bg-dark-800 text-dark-300 hover:text-white hover:bg-dark-700">
            <ChevronRight className="w-5 h-5" />
          </button>
          <button onClick={() => { setYear(now.getFullYear()); setMonth(now.getMonth()); }}
            className="px-3 py-1.5 text-sm bg-dark-700 text-dark-300 rounded-lg hover:bg-dark-600 hover:text-white">
            Heute
          </button>
        </div>
      </div>

      {/* Legend */}
      <div className="flex flex-wrap items-center gap-3 text-xs text-dark-400">
        <span className="flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm bg-amber-500/30 border border-amber-500/50 inline-block" />Offene Aufgabe</span>
        <span className="flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm bg-emerald-500/30 border border-emerald-500/50 inline-block" />Erledigte Aufgabe</span>
        <span className="flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm bg-primary-500/30 border border-primary-500/50 inline-block" />Ankunft (geplant)</span>
        <span className="flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm bg-teal-500/30 border border-teal-500/50 inline-block" />Ankunft (eingetroffen)</span>
        {(Object.keys(ENTRY_TYPE_LABELS) as CalendarEntryType[]).map(t => (
          <span key={t} className="flex items-center gap-1.5">
            <span className={`w-3 h-3 rounded-sm inline-block ${ENTRY_TYPE_COLORS[t].split(' ')[0]}`} />
            {ENTRY_TYPE_LABELS[t]}
          </span>
        ))}
      </div>

      {isLoading ? (
        <div className="text-dark-400">Lädt…</div>
      ) : (
        <div className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
          {/* Weekday header */}
          <div className="grid grid-cols-7 border-b border-dark-800">
            {DAYS_DE.map(d => (
              <div key={d} className={`py-2 text-center text-xs font-medium ${d === 'Sa' || d === 'So' ? 'text-dark-500' : 'text-dark-400'}`}>{d}</div>
            ))}
          </div>

          {/* Grid */}
          <div className="grid grid-cols-7">
            {cells.map((day, i) => {
              if (!day) return <div key={`e${i}`} className="min-h-[110px] border-b border-r border-dark-800/50 bg-dark-950/30" />;
              const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
              const ev = eventMap[dateStr];
              const isToday   = dateStr === todayStr;
              const isWeekend = (i % 7) >= 5;

              return (
                <div key={dateStr}
                  className={`min-h-[110px] border-b border-r border-dark-800/50 p-1 group ${isWeekend ? 'bg-dark-950/20' : ''}`}>
                  <div className="flex items-center justify-between mb-0.5">
                    <span className={`text-xs font-medium w-6 h-6 flex items-center justify-center rounded-full
                      ${isToday ? 'bg-primary-500 text-white' : isWeekend ? 'text-dark-500' : 'text-dark-400'}`}>
                      {day}
                    </span>
                    <button
                      onClick={() => openNewEntry(dateStr)}
                      className="opacity-0 group-hover:opacity-100 p-0.5 text-dark-500 hover:text-primary-400 transition-opacity"
                      title="Termin hinzufügen"
                    >
                      <Plus className="w-3.5 h-3.5" />
                    </button>
                  </div>

                  {ev && (
                    <div className="space-y-0.5">
                      {ev.tasks.map(t => (
                        <Link key={t.id} to={`/procurement/orders/${t.orderId}`}
                          className={`block text-xs px-1.5 py-0.5 rounded truncate leading-tight
                            ${t.doneAt ? 'bg-emerald-500/20 text-emerald-400 line-through' : 'bg-amber-500/20 text-amber-300'}`}
                          title={`${t.orderNumber}: ${t.text}`}>
                          <CheckSquare className="w-2.5 h-2.5 inline mr-0.5 flex-shrink-0" />
                          {t.text}
                        </Link>
                      ))}
                      {ev.arrivals.map(a => (
                        <div key={a.orderId}
                          className={`text-xs px-1.5 py-0.5 rounded leading-tight flex items-center gap-0.5
                            ${a.actualDate ? 'bg-teal-500/20 text-teal-400' : 'bg-primary-500/20 text-primary-300'}`}>
                          <button
                            onClick={() => setFreightModalArrival(a)}
                            className="flex-1 flex items-center gap-0.5 min-w-0 truncate text-left hover:opacity-80"
                            title={`${a.orderNumber}${a.supplierName ? ' – ' + a.supplierName : ''} — Datum bearbeiten`}
                          >
                            <Ship className="w-2.5 h-2.5 flex-shrink-0" />
                            <span className="truncate">{a.orderNumber}{a.supplierName ? ` – ${a.supplierName}` : ''}</span>
                          </button>
                          <Link
                            to={`/procurement/orders/${a.orderId}`}
                            className="flex-shrink-0 hover:opacity-80"
                            title="Zur Bestellung"
                            onClick={e => e.stopPropagation()}
                          >
                            <ExternalLink className="w-2.5 h-2.5" />
                          </Link>
                        </div>
                      ))}
                      {ev.entries.map(e => (
                        <div key={e.id}
                          className={`text-xs px-1.5 py-0.5 rounded truncate leading-tight flex items-center gap-1 group/entry ${ENTRY_TYPE_COLORS[e.entryType]}`}
                          title={e.notes ? `${e.title}: ${e.notes}` : e.title}>
                          <span className="flex-1 truncate">{e.title}</span>
                          <button
                            onClick={() => { setEditingEntry(e); setShowForm(true); }}
                            className="opacity-0 group-hover/entry:opacity-100 flex-shrink-0 hover:text-white"
                          >
                            <Pencil className="w-2.5 h-2.5" />
                          </button>
                          <button
                            onClick={() => { if (confirm(`„${e.title}" löschen?`)) deleteMut.mutate(e.id); }}
                            className="opacity-0 group-hover/entry:opacity-100 flex-shrink-0 hover:text-red-300"
                          >
                            <Trash2 className="w-2.5 h-2.5" />
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Detail lists */}
      {(tasks.length > 0 || arrivals.length > 0 || entries.length > 0) && (
        <div className="grid md:grid-cols-3 gap-6">
          {tasks.length > 0 && (
            <div className="bg-dark-900/50 border border-dark-800 rounded-xl p-4 space-y-2">
              <h3 className="text-sm font-semibold text-white mb-3">Aufgaben</h3>
              {tasks.map(t => (
                <div key={t.id} className="flex items-start gap-2 text-sm">
                  <span className={`mt-0.5 flex-shrink-0 ${t.doneAt ? 'text-emerald-400' : 'text-amber-400'}`}>
                    <CheckSquare className="w-4 h-4" />
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className={`${t.doneAt ? 'line-through text-dark-500' : 'text-white'} truncate`}>{t.text}</div>
                    <div className="text-dark-400 text-xs">
                      <Link to={`/procurement/orders/${t.orderId}`} className="hover:text-primary-400">{t.orderNumber}</Link>
                      {' · '}{new Date(t.dueDate + 'T00:00:00').toLocaleDateString('de-DE')}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {arrivals.length > 0 && (
            <div className="bg-dark-900/50 border border-dark-800 rounded-xl p-4 space-y-2">
              <h3 className="text-sm font-semibold text-white mb-3">Warenankünfte</h3>
              {arrivals.map(a => (
                <div key={a.orderId} className="flex items-start gap-2 text-sm">
                  <span className={`mt-0.5 flex-shrink-0 ${a.actualDate ? 'text-teal-400' : 'text-primary-400'}`}>
                    <Ship className="w-4 h-4" />
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="text-white">
                      <Link to={`/procurement/orders/${a.orderId}`} className="hover:text-primary-400">{a.orderNumber}</Link>
                      {a.supplierName && <span className="text-dark-400"> – {a.supplierName}</span>}
                    </div>
                    <div className="text-dark-400 text-xs">
                      {a.actualDate
                        ? `Eingetroffen: ${new Date(a.actualDate + 'T00:00:00').toLocaleDateString('de-DE')}`
                        : `ETA: ${new Date(a.estimatedDate + 'T00:00:00').toLocaleDateString('de-DE')}`}
                    </div>
                  </div>
                  <button
                    onClick={() => setFreightModalArrival(a)}
                    className="p-1 flex-shrink-0 text-dark-500 hover:text-primary-400"
                    title="Frachtdatum bearbeiten"
                  >
                    <Pencil className="w-3.5 h-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}

          {entries.length > 0 && (
            <div className="bg-dark-900/50 border border-dark-800 rounded-xl p-4 space-y-2">
              <h3 className="text-sm font-semibold text-white mb-3">Einträge</h3>
              {entries.map(e => (
                <div key={e.id} className="flex items-start gap-2 text-sm">
                  <span className={`mt-0.5 flex-shrink-0 w-2 h-2 rounded-full mt-2 flex-shrink-0 ${ENTRY_TYPE_COLORS[e.entryType].split(' ')[0]}`} />
                  <div className="flex-1 min-w-0">
                    <div className="text-white truncate">{e.title}</div>
                    <div className="text-dark-400 text-xs">
                      {ENTRY_TYPE_LABELS[e.entryType]} · {new Date(e.date).toLocaleDateString('de-DE')}
                    </div>
                    {e.notes && <div className="text-dark-500 text-xs truncate">{e.notes}</div>}
                  </div>
                  <div className="flex items-center gap-1 flex-shrink-0">
                    <button onClick={() => { setEditingEntry(e); setShowForm(true); }}
                      className="p-1 text-dark-500 hover:text-primary-400"><Pencil className="w-3.5 h-3.5" /></button>
                    <button onClick={() => { if (confirm(`„${e.title}" löschen?`)) deleteMut.mutate(e.id); }}
                      className="p-1 text-dark-500 hover:text-red-400"><Trash2 className="w-3.5 h-3.5" /></button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Entry form modal */}
      {(showForm || editingEntry) && (
        <EntryForm
          initial={editingEntry ?? undefined}
          defaultDate={formDefaultDate}
          onSave={handleSave}
          onClose={() => { setShowForm(false); setEditingEntry(null); }}
          saving={saving}
        />
      )}

      {/* Freight date modal */}
      {freightModalArrival && (
        <FreightDateModal
          arrival={freightModalArrival}
          onClose={() => setFreightModalArrival(null)}
        />
      )}
    </div>
  );
}
