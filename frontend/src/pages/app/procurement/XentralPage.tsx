import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link2, CheckCircle, XCircle, RefreshCw, Settings2, Clock, AlertCircle, Building2, Download, Copy, Check } from 'lucide-react';
import { toast } from 'sonner';
import { xentralApi, type XentralConfig, type XentralSyncLog, type XentralInstanceInfo, type XentralSyncEntity } from '../../../api/xentral';
import { useTenant } from '../../../contexts/TenantContext';

const inputCls = 'w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500';
const labelCls = 'block text-xs text-dark-400 mb-1';

interface EntityDef {
  key: keyof XentralConfig;
  label: string;
  syncKey: XentralSyncEntity;
  lastKey: keyof XentralConfig;
  group: 'inbound' | 'outbound';
  description?: string;
}

const ENTITIES: EntityDef[] = [
  // Inbound: Xentral → LastSaaS
  { key: 'syncProducts',       label: 'Artikel',           syncKey: 'products',        lastKey: 'lastSyncProducts',       group: 'inbound', description: 'Artikel, EANs, Freefields' },
  { key: 'syncCustomers',      label: 'Kunden',            syncKey: 'customers',       lastKey: 'lastSyncCustomers',      group: 'inbound' },
  { key: 'syncSuppliers',      label: 'Lieferanten',       syncKey: 'suppliers',       lastKey: 'lastSyncSuppliers',      group: 'inbound' },
  { key: 'syncOrders',         label: 'Bestellungen',      syncKey: 'orders',          lastKey: 'lastSyncOrders',         group: 'inbound', description: 'Einmalig-Import (Historien-Migration)' },
  { key: 'syncSalesOrders',    label: 'Verkaufsaufträge',  syncKey: 'sales_orders',    lastKey: 'lastSyncSalesOrders',    group: 'inbound', description: 'Für Dispositionsplanung' },
  { key: 'syncPurchasePrices', label: 'EK-Preislisten',    syncKey: 'purchase_prices', lastKey: 'lastSyncPurchasePrices', group: 'inbound' },
  { key: 'syncSalesPrices',    label: 'VK-Preislisten',    syncKey: 'sales_prices',    lastKey: 'lastSyncSalesPrices',    group: 'inbound' },
  { key: 'syncWarehouses',     label: 'Lagerorte',         syncKey: 'warehouses',      lastKey: 'lastSyncWarehouses',     group: 'inbound', description: 'Lagerorte + Lagerplätze' },
  { key: 'syncStocks',         label: 'Lagerbestände',     syncKey: 'stocks',          lastKey: 'lastSyncStocks',         group: 'inbound', description: 'Mengen, MHD, Serien- und Chargennummern' },
  // Outbound: LastSaaS → Xentral
  { key: 'pushOrdersToXentral', label: 'Bestellungen →', syncKey: 'push_orders', lastKey: 'lastPushOrders', group: 'outbound', description: 'Bestellungen nach Xentral pushen' },
];

const STATUS_COLORS: Record<string, string> = {
  success: 'bg-green-500/20 text-green-300',
  partial: 'bg-amber-500/20 text-amber-300',
  error:   'bg-red-500/20 text-red-300',
  running: 'bg-blue-500/20 text-blue-300',
  skipped: 'bg-dark-700 text-dark-400',
};

function fmt(iso?: string) {
  if (!iso) return '—';
  return new Date(iso).toLocaleString('de-DE', { dateStyle: 'short', timeStyle: 'short' });
}

export default function XentralPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;

  const { data: cfg, isLoading: cfgLoading } = useQuery({
    queryKey: ['xentral-config'],
    queryFn: xentralApi.getConfig,
    enabled,
    throwOnError: false,
  });
  const { data: logs = [], isLoading: logsLoading } = useQuery({
    queryKey: ['xentral-logs'],
    queryFn: xentralApi.getLogs,
    enabled,
    throwOnError: false,
    refetchInterval: 15000,
  });

  // Form state for config editing
  const [form, setForm] = useState<Partial<XentralConfig & { apiToken: string }>>({});
  const [editingToken, setEditingToken] = useState(false);
  const [testing, setTesting] = useState(false);
  const [liveInstanceInfo, setLiveInstanceInfo] = useState<XentralInstanceInfo | null>(null);
  const [copiedToken, setCopiedToken] = useState(false);
  const [syncingEntity, setSyncingEntity] = useState<string | null>(null);
  const [syncingAll, setSyncingAll] = useState(false);
  const [importingAccount, setImportingAccount] = useState(false);

  // Merge server config into local form state once loaded
  const merged: Partial<XentralConfig & { apiToken: string }> = {
    baseUrl: '',
    enabled: true,
    syncIntervalH: 0,
    syncProducts: true,
    syncCustomers: true,
    syncSuppliers: true,
    syncOrders: false,
    syncSalesOrders: false,
    syncPurchasePrices: false,
    syncSalesPrices: false,
    syncWarehouses: false,
    syncStocks: false,
    pushProductsToXentral: false,
    pushOrdersToXentral: false,
    ...cfg,
    ...form,
  };

  const saveMut = useMutation({
    mutationFn: () => xentralApi.saveConfig({ ...merged as any }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['xentral-config'] });
      setForm({});
      setEditingToken(false);
      toast.success('Einstellungen gespeichert');
    },
    onError: () => toast.error('Fehler beim Speichern'),
  });

  const handleTest = async () => {
    setTesting(true);
    try {
      // Save first if there are unsaved changes
      if (Object.keys(form).length > 0) {
        await xentralApi.saveConfig({ ...merged as any });
        qc.invalidateQueries({ queryKey: ['xentral-config'] });
      }
      const result = await xentralApi.testConnection();
      if (result.ok) {
        toast.success('Verbindung erfolgreich');
        if (result.instanceInfo) setLiveInstanceInfo(result.instanceInfo);
        qc.invalidateQueries({ queryKey: ['xentral-config'] });
      } else {
        toast.error(`Verbindung fehlgeschlagen: ${result.error}`);
      }
    } catch {
      toast.error('Verbindungstest fehlgeschlagen');
    } finally {
      setTesting(false);
    }
  };

  const handleSyncEntity = async (entity: XentralSyncEntity) => {
    setSyncingEntity(entity);
    try {
      await xentralApi.syncEntity(entity);
      qc.invalidateQueries({ queryKey: ['xentral-logs'] });
      qc.invalidateQueries({ queryKey: ['xentral-config'] });
      toast.success(`Sync abgeschlossen`);
    } catch {
      toast.error('Sync fehlgeschlagen');
    } finally {
      setSyncingEntity(null);
    }
  };

  const handleSyncAll = async () => {
    setSyncingAll(true);
    try {
      await xentralApi.syncAll();
      qc.invalidateQueries({ queryKey: ['xentral-logs'] });
      qc.invalidateQueries({ queryKey: ['xentral-config'] });
      toast.success('Alle Sync-Läufe abgeschlossen');
    } catch {
      toast.error('Sync fehlgeschlagen');
    } finally {
      setSyncingAll(false);
    }
  };

  const handleImportAccount = async () => {
    setImportingAccount(true);
    try {
      await xentralApi.importAccount();
      toast.success('Firmenprofil übernommen');
    } catch {
      toast.error('Import fehlgeschlagen');
    } finally {
      setImportingAccount(false);
    }
  };

  const upd = (key: string, value: unknown) => setForm(f => ({ ...f, [key]: value }));

  if (cfgLoading) return <div className="text-dark-400 py-8">Lädt…</div>;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link2 className="w-6 h-6 text-primary-400" />
        <div>
          <h1 className="text-2xl font-bold text-white">Xentral ERP</h1>
          <p className="text-dark-400 mt-1">Bidirektionale Synchronisierung mit Xentral ERP</p>
        </div>
      </div>

      {/* Connection config */}
      <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-5">
        <div className="flex items-center gap-2">
          <Settings2 className="w-4 h-4 text-dark-400" />
          <h2 className="text-sm font-semibold text-white uppercase tracking-wide">Verbindung</h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className={labelCls}>Xentral-URL *</label>
            <input
              type="url"
              placeholder="https://meinefirma.xentral.biz"
              value={merged.baseUrl ?? ''}
              onChange={e => upd('baseUrl', e.target.value)}
              className={inputCls}
            />
          </div>
          <div>
            <label className={labelCls}>API-Token</label>
            {editingToken || !cfg?.apiTokenMask ? (
              <input
                type="password"
                placeholder="Neuen Token eingeben"
                value={merged.apiToken ?? ''}
                onChange={e => upd('apiToken', e.target.value)}
                className={inputCls}
              />
            ) : (
              <div className="flex items-center gap-2">
                <span className="flex-1 px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-dark-400 text-sm font-mono">
                  {cfg.apiTokenMask}
                </span>
                <button onClick={() => setEditingToken(true)}
                  className="px-3 py-2 bg-dark-700 text-dark-300 rounded-lg hover:bg-dark-600 text-sm">
                  Ändern
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Webhook URL — shown once a webhookToken exists (after first save) */}
        {cfg?.webhookToken && (() => {
          const webhookUrl = `${window.location.origin}/api/procurement/integrations/xentral/webhook/${cfg.webhookToken}`;
          return (
            <div>
              <label className={labelCls}>Webhook-URL (für Xentral)</label>
              <div className="flex items-center gap-2">
                <span className="flex-1 px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-dark-400 text-xs font-mono truncate select-all">
                  {webhookUrl}
                </span>
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(webhookUrl);
                    setCopiedToken(true);
                    setTimeout(() => setCopiedToken(false), 2000);
                  }}
                  title="Kopieren"
                  className="px-3 py-2 bg-dark-700 text-dark-300 rounded-lg hover:bg-dark-600 text-sm flex items-center gap-1.5">
                  {copiedToken ? <Check className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4" />}
                </button>
              </div>
              <p className="mt-1 text-xs text-dark-500">
                Trage diese URL in Xentral als Webhook ein, damit Änderungen in Echtzeit übertragen werden.
              </p>
            </div>
          );
        })()}

        <div className="flex items-center gap-3">
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={merged.enabled ?? true}
              onChange={e => upd('enabled', e.target.checked)}
              className="w-4 h-4 accent-primary-500" />
            <span className="text-sm text-white">Integration aktiv</span>
          </label>
        </div>

        <div className="flex gap-3">
          <button
            onClick={() => saveMut.mutate()}
            disabled={saveMut.isPending}
            className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 text-sm">
            {saveMut.isPending ? 'Speichert…' : 'Speichern'}
          </button>
          <button
            onClick={handleTest}
            disabled={testing}
            className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600 disabled:opacity-50 text-sm flex items-center gap-2">
            {testing ? <RefreshCw className="w-4 h-4 animate-spin" /> : null}
            {testing ? 'Teste…' : 'Verbindung testen'}
          </button>
        </div>

        {/* Instance info card — shown after successful test or when cached */}
        {(() => {
          const info = liveInstanceInfo ?? cfg?.instanceInfo;
          if (!info?.companyName) return null;
          return (
            <div className="mt-2 bg-dark-800/60 border border-dark-700 rounded-lg p-4 space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Building2 className="w-4 h-4 text-primary-400" />
                  <span className="text-sm font-semibold text-white">{info.companyName}</span>
                  {info.version && (
                    <span className="px-1.5 py-0.5 rounded text-xs bg-dark-700 text-dark-400 font-mono">
                      Xentral {info.version}{info.edition ? ' ' + info.edition : ''}
                    </span>
                  )}
                </div>
                <button
                  onClick={handleImportAccount}
                  disabled={importingAccount}
                  title="Firmenstammdaten als eigenes Firmenprofil übernehmen"
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-dark-700 text-dark-300 rounded-lg hover:bg-dark-600 hover:text-white disabled:opacity-50 text-xs">
                  {importingAccount ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Download className="w-3.5 h-3.5" />}
                  Als Firmenprofil übernehmen
                </button>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-x-6 gap-y-1 text-xs text-dark-400">
                {info.street && <span>{info.street}, {info.zip} {info.city}{info.country ? ', ' + info.country : ''}</span>}
                {info.email && <span>✉ {info.email}</span>}
                {info.phone && <span>☎ {info.phone}</span>}
                {info.website && <span>🌐 {info.website}</span>}
                {info.taxId && <span>StNr: {info.taxId}</span>}
                {info.vatId && <span>USt-IdNr: {info.vatId}</span>}
                {info.currency && <span>Währung: {info.currency}</span>}
                {info.language && <span>Sprache: {info.language}</span>}
              </div>
              {info.fetchedAt && (
                <p className="text-xs text-dark-600">Zuletzt abgerufen: {fmt(info.fetchedAt)}</p>
              )}
            </div>
          );
        })()}
      </div>

      {/* Entity sync settings */}
      <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Clock className="w-4 h-4 text-dark-400" />
            <h2 className="text-sm font-semibold text-white uppercase tracking-wide">Synchronisierung</h2>
          </div>
          <button
            onClick={handleSyncAll}
            disabled={syncingAll || !merged.enabled}
            className="flex items-center gap-2 px-3 py-1.5 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 text-sm">
            {syncingAll ? <RefreshCw className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
            Alles synchronisieren
          </button>
        </div>

        {/* Auto-sync interval */}
        <div className="max-w-xs">
          <label className={labelCls}>Auto-Sync-Intervall</label>
          <select
            value={merged.syncIntervalH ?? 0}
            onChange={e => upd('syncIntervalH', parseInt(e.target.value))}
            className={inputCls}>
            <option value={0}>Manuell (kein Auto-Sync)</option>
            <option value={1}>Stündlich</option>
            <option value={6}>Alle 6 Stunden</option>
            <option value={12}>Alle 12 Stunden</option>
            <option value={24}>Täglich</option>
          </select>
        </div>

        {/* Per-entity rows — grouped inbound / outbound */}
        {(['inbound', 'outbound'] as const).map(group => {
          const groupEntities = ENTITIES.filter(e => e.group === group);
          return (
            <div key={group} className="space-y-1">
              <p className="text-xs text-dark-500 font-medium uppercase tracking-wider pb-1">
                {group === 'inbound' ? '← Xentral → LastSaaS' : '→ LastSaaS → Xentral'}
              </p>
              {groupEntities.map(({ key, label, syncKey, lastKey, description }) => (
                <div key={key} className="flex items-center gap-4 py-2.5 border-b border-dark-800/60 last:border-0">
                  <label className="flex items-center gap-2 cursor-pointer w-44 flex-shrink-0">
                    <input type="checkbox" checked={merged[key] as boolean ?? false}
                      onChange={e => upd(key, e.target.checked)}
                      className="w-4 h-4 accent-primary-500" />
                    <span className="text-sm text-white">{label}</span>
                  </label>
                  <span className="text-xs text-dark-500 flex-1 hidden sm:block">
                    {description && <span className="mr-2 text-dark-600">{description} ·</span>}
                    Letzter Sync: {fmt(cfg?.[lastKey] as string | undefined)}
                  </span>
                  <button
                    onClick={() => handleSyncEntity(syncKey)}
                    disabled={syncingEntity === syncKey || !merged.enabled || !merged[key]}
                    className="flex items-center gap-1.5 px-2.5 py-1 bg-dark-700 text-dark-300 rounded hover:bg-dark-600 hover:text-white disabled:opacity-40 text-xs">
                    {syncingEntity === syncKey
                      ? <RefreshCw className="w-3 h-3 animate-spin" />
                      : <RefreshCw className="w-3 h-3" />}
                    Sync
                  </button>
                </div>
              ))}
            </div>
          );
        })}

        {/* Save sync settings */}
        <button
          onClick={() => saveMut.mutate()}
          disabled={saveMut.isPending}
          className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600 disabled:opacity-50 text-sm">
          Einstellungen speichern
        </button>
      </div>

      {/* Sync log */}
      <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
        <h2 className="text-sm font-semibold text-white uppercase tracking-wide">Sync-Protokoll</h2>

        {logsLoading ? (
          <div className="text-dark-400 text-sm">Lädt…</div>
        ) : logs.length === 0 ? (
          <p className="text-dark-500 text-sm">Noch keine Sync-Läufe vorhanden.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead className="border-b border-dark-700">
                <tr>
                  <th className="text-left py-2 pr-4 text-dark-400 font-medium">Zeitpunkt</th>
                  <th className="text-left py-2 pr-4 text-dark-400 font-medium">Entität</th>
                  <th className="text-center py-2 pr-4 text-dark-400 font-medium">Status</th>
                  <th className="text-right py-2 pr-4 text-dark-400 font-medium">Erstellt</th>
                  <th className="text-right py-2 pr-4 text-dark-400 font-medium">Aktualisiert</th>
                  <th className="text-right py-2 text-dark-400 font-medium">Übersprungen</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-dark-800/50">
                {(logs as XentralSyncLog[]).map(log => (
                  <tr key={log.id} className="hover:bg-dark-800/20">
                    <td className="py-2 pr-4 text-dark-300 font-mono">{fmt(log.startedAt)}</td>
                    <td className="py-2 pr-4 text-white capitalize">{log.entity}</td>
                    <td className="py-2 pr-4 text-center">
                      <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${STATUS_COLORS[log.status] ?? 'bg-dark-700 text-dark-400'}`}>
                        {log.status === 'success' && <CheckCircle className="w-3 h-3 inline mr-1" />}
                        {log.status === 'error' && <XCircle className="w-3 h-3 inline mr-1" />}
                        {log.status === 'partial' && <AlertCircle className="w-3 h-3 inline mr-1" />}
                        {log.status}
                      </span>
                    </td>
                    <td className="py-2 pr-4 text-right text-dark-300">{log.created}</td>
                    <td className="py-2 pr-4 text-right text-dark-300">{log.updated}</td>
                    <td className="py-2 text-right text-dark-500">{log.skipped}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Error details for last failed run */}
        {logs.some((l: XentralSyncLog) => l.errors && l.errors.length > 0) && (
          <details className="mt-2">
            <summary className="text-xs text-dark-500 cursor-pointer hover:text-dark-300">Fehlerdetails anzeigen</summary>
            <div className="mt-2 space-y-2">
              {(logs as XentralSyncLog[])
                .filter(l => l.errors && l.errors.length > 0)
                .slice(0, 5)
                .map(l => (
                  <div key={l.id} className="bg-red-500/5 border border-red-500/20 rounded p-2">
                    <p className="text-xs text-red-300 font-medium mb-1">{l.entity} — {fmt(l.startedAt)}</p>
                    <ul className="space-y-0.5">
                      {l.errors!.map((e, i) => (
                        <li key={i} className="text-xs text-red-400 font-mono">{e}</li>
                      ))}
                    </ul>
                  </div>
                ))}
            </div>
          </details>
        )}
      </div>
    </div>
  );
}
