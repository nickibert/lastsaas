import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Pencil, Settings, ClipboardList, BarChart2 } from 'lucide-react';
import { toast } from 'sonner';
import { taskTemplatesApi, procurementConfigApi, type TaskTemplate, type EkDbMethode } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';

const inputCls = 'w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500';
const labelCls = 'block text-xs text-dark-400 mb-1';

const phaseLabels: Record<number, string> = {
  1: 'Phase 1 – Bestellung aufgegeben',
  2: 'Phase 2 – Ware verschifft',
};

function TaskTemplateSection() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<TaskTemplate | null>(null);
  const [form, setForm] = useState<Partial<TaskTemplate>>({ text: '', daysAfter: 0, phase: 1 });

  const { data: templates = [], isLoading } = useQuery({
    queryKey: ['task-templates'],
    queryFn: taskTemplatesApi.list,
    enabled,
    throwOnError: false,
  });

  const createMut = useMutation({
    mutationFn: taskTemplatesApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['task-templates'] }); setShowForm(false); resetForm(); toast.success('Vorlage angelegt'); },
    onError: () => toast.error('Fehler'),
  });
  const updateMut = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<TaskTemplate> }) => taskTemplatesApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['task-templates'] }); setEditing(null); setShowForm(false); resetForm(); toast.success('Gespeichert'); },
    onError: () => toast.error('Fehler'),
  });
  const deleteMut = useMutation({
    mutationFn: taskTemplatesApi.delete,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['task-templates'] }); toast.success('Gelöscht'); },
  });

  const resetForm = () => setForm({ text: '', daysAfter: 0, phase: 1 });
  const submit = () => {
    if (!form.text) return;
    if (editing) updateMut.mutate({ id: editing.id, data: form });
    else createMut.mutate(form);
  };
  const startEdit = (t: TaskTemplate) => { setEditing(t); setForm({ ...t }); setShowForm(true); };

  const grouped = [1, 2].map(phase => ({
    phase,
    items: templates.filter(t => t.phase === phase),
  }));

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <ClipboardList className="w-5 h-5 text-primary-400" />
          <h2 className="text-lg font-semibold text-white">Aufgaben-Vorlagen</h2>
        </div>
        <button onClick={() => { resetForm(); setEditing(null); setShowForm(true); }}
          className="flex items-center gap-2 px-3 py-1.5 bg-primary-500 text-white rounded-lg hover:bg-primary-600 text-sm">
          <Plus className="w-4 h-4" /> Vorlage anlegen
        </button>
      </div>
      <p className="text-sm text-dark-400">
        Vorlagen werden automatisch als Aufgaben erstellt, wenn eine Bestellung in die entsprechende Phase wechselt.
      </p>

      {showForm && (
        <div className="bg-dark-800/50 border border-dark-700 rounded-xl p-4 space-y-4">
          <h3 className="text-sm font-semibold text-white">{editing ? 'Vorlage bearbeiten' : 'Neue Vorlage'}</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="md:col-span-2">
              <label className={labelCls}>Aufgabentext *</label>
              <input type="text" value={form.text ?? ''} onChange={e => setForm(f => ({ ...f, text: e.target.value }))}
                placeholder="z.B. Zahlungsbestätigung einholen" className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Phase</label>
              <select value={form.phase ?? 1} onChange={e => setForm(f => ({ ...f, phase: parseInt(e.target.value) as 1 | 2 }))}
                className={inputCls}>
                <option value={1}>Phase 1 – Bestellung aufgegeben</option>
                <option value={2}>Phase 2 – Ware verschifft</option>
              </select>
            </div>
            <div>
              <label className={labelCls}>Fällig nach (Tage)</label>
              <input type="number" min="0" value={form.daysAfter ?? 0}
                onChange={e => setForm(f => ({ ...f, daysAfter: parseInt(e.target.value) }))}
                className={inputCls} />
            </div>
          </div>
          <div className="flex gap-3">
            <button onClick={submit} disabled={!form.text || createMut.isPending || updateMut.isPending}
              className="px-3 py-1.5 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 text-sm">
              Speichern
            </button>
            <button onClick={() => { setShowForm(false); setEditing(null); resetForm(); }}
              className="px-3 py-1.5 bg-dark-700 text-white rounded-lg hover:bg-dark-600 text-sm">
              Abbrechen
            </button>
          </div>
        </div>
      )}

      {isLoading ? <div className="text-dark-400 text-sm">Lädt...</div> : (
        <div className="space-y-4">
          {grouped.map(({ phase, items }) => (
            <div key={phase} className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
              <div className="px-4 py-2 bg-dark-800/50 border-b border-dark-800">
                <span className="text-sm font-medium text-dark-300">{phaseLabels[phase]}</span>
              </div>
              {items.length === 0 ? (
                <div className="px-4 py-4 text-sm text-dark-500">Keine Vorlagen für diese Phase</div>
              ) : (
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-dark-400 text-xs">
                      <th className="text-left px-4 py-2">Aufgabentext</th>
                      <th className="text-left px-4 py-2">Fällig nach</th>
                      <th className="px-4 py-2"></th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-dark-800/50">
                    {items.map(t => (
                      <tr key={t.id} className="hover:bg-dark-800/30">
                        <td className="px-4 py-2.5 text-white">{t.text}</td>
                        <td className="px-4 py-2.5 text-dark-300">
                          {t.daysAfter === 0 ? 'sofort' : `${t.daysAfter} Tag${t.daysAfter !== 1 ? 'e' : ''}`}
                        </td>
                        <td className="px-4 py-2.5 text-right">
                          <div className="flex items-center justify-end gap-1">
                            <button onClick={() => startEdit(t)} className="p-1 text-dark-500 hover:text-primary-400">
                              <Pencil className="w-3.5 h-3.5" />
                            </button>
                            <button onClick={() => { if (confirm('Vorlage löschen?')) deleteMut.mutate(t.id); }}
                              className="p-1 text-dark-500 hover:text-red-400">
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

const EK_METHOD_OPTIONS: { value: EkDbMethode; label: string; description: string }[] = [
  { value: 'last',        label: 'Letzter EK',        description: 'Zuletzt gespeicherter EK-Preis des Artikels (Product.LastEK)' },
  { value: 'fifo',        label: 'FIFO',               description: 'First In, First Out – älteste Wareneingänge gelten als zuerst verbraucht' },
  { value: 'lifo',        label: 'LIFO',               description: 'Last In, First Out – neueste Wareneingänge gelten als zuerst verbraucht' },
  { value: 'weighted_avg',label: 'Ø Durchschnitt',     description: 'Gewogener Durchschnitt über alle jemals eingegangenen Lose' },
];

function EkMethodSection() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;

  const { data: cfg, isLoading } = useQuery({
    queryKey: ['procurement-config'],
    queryFn: procurementConfigApi.get,
    enabled,
    throwOnError: false,
  });

  const updateMut = useMutation({
    mutationFn: (method: EkDbMethode) => procurementConfigApi.update({ ekDbMethode: method }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['procurement-config'] }); toast.success('Einstellung gespeichert'); },
    onError: () => toast.error('Fehler beim Speichern'),
  });

  const active = cfg?.ekDbMethode ?? 'fifo';

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <BarChart2 className="w-5 h-5 text-primary-400" />
        <h2 className="text-lg font-semibold text-white">EK-Methode für Deckungsbeitrag</h2>
      </div>
      <p className="text-sm text-dark-400">
        Legt fest, welcher Einstandspreis für die Deckungsbeitragsberechnung herangezogen wird.
        Im Artikel-Tab "Lagerbewertung" werden alle Methoden zum Vergleich angezeigt.
      </p>

      {isLoading ? (
        <div className="text-dark-400 text-sm">Lädt…</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {EK_METHOD_OPTIONS.map(opt => (
            <button
              key={opt.value}
              onClick={() => { if (opt.value !== active) updateMut.mutate(opt.value); }}
              disabled={updateMut.isPending}
              className={`text-left px-4 py-3 rounded-lg border transition-colors ${
                opt.value === active
                  ? 'border-primary-500 bg-primary-500/10 text-white'
                  : 'border-dark-700 bg-dark-800/50 text-dark-300 hover:border-dark-600 hover:text-white'
              }`}
            >
              <div className="flex items-center gap-2">
                <span className={`w-3 h-3 rounded-full border-2 flex-shrink-0 ${opt.value === active ? 'border-primary-400 bg-primary-400' : 'border-dark-500'}`} />
                <span className="font-medium text-sm">{opt.label}</span>
                {opt.value === active && (
                  <span className="ml-auto text-xs px-1.5 py-0.5 rounded bg-primary-500/20 text-primary-300">aktiv</span>
                )}
              </div>
              <p className="text-xs text-dark-500 mt-1 ml-5">{opt.description}</p>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

export default function ProcurementConfigPage() {
  return (
    <div className="space-y-8">
      <div className="flex items-center gap-3">
        <Settings className="w-6 h-6 text-dark-400" />
        <div>
          <h1 className="text-2xl font-bold text-white">Konfiguration</h1>
          <p className="text-dark-400 mt-1">Aufgaben-Vorlagen und weitere Einstellungen</p>
        </div>
      </div>

      <EkMethodSection />
      <TaskTemplateSection />
    </div>
  );
}
