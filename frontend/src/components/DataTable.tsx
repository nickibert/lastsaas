/**
 * DataTable – reusable table component with:
 *  - Configurable column visibility (persisted in localStorage)
 *  - Row selection + bulk-action bar
 *  - Inline cell editing (click a cell to edit, Enter/Tab to confirm, Esc to cancel)
 *  - Optional search bar and filter slot
 *  - Sorting (click column header)
 */
import { useState, useEffect, useRef, useCallback } from 'react';
import { Columns3, X, ChevronUp, ChevronDown, ChevronsUpDown, Trash2, Check, SlidersHorizontal } from 'lucide-react';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type SortDir = 'asc' | 'desc';

export interface ColumnDef<T> {
  /** Unique key, also used for localStorage persistence. */
  key: string;
  header: string;
  /** Render a read-only cell. */
  render: (row: T) => React.ReactNode;
  /**
   * If provided, clicking the cell enters edit mode and renders this.
   * Return null to make the column non-editable.
   */
  renderEdit?: (row: T, value: string, onChange: (v: string) => void) => React.ReactNode;
  /** Extract the initial string value for editing. */
  getValue?: (row: T) => string;
  /** Called when the user commits a cell edit. */
  onEdit?: (row: T, newValue: string) => void;
  /** Width hint, e.g. "w-32" or "min-w-[120px]". */
  width?: string;
  sortable?: boolean;
  /** Default visibility; defaults to true. */
  defaultVisible?: boolean;
}

export interface BulkAction<T> {
  label: string;
  icon?: React.ReactNode;
  variant?: 'default' | 'danger';
  onClick: (selected: T[]) => void;
}

export interface DataTableProps<T> {
  /** Stable key for persisting column visibility preferences. */
  tableKey: string;
  columns: ColumnDef<T>[];
  data: T[];
  getRowId: (row: T) => string;
  /** Optional search bar value (controlled externally). */
  searchValue?: string;
  onSearchChange?: (v: string) => void;
  searchPlaceholder?: string;
  /** Slot for additional filter controls, rendered next to the search bar. */
  filterSlot?: React.ReactNode;
  bulkActions?: BulkAction<T>[];
  /** Called when user clicks a row (outside of an editable cell). */
  onRowClick?: (row: T) => void;
  emptyMessage?: string;
  isLoading?: boolean;
  /** Current sort key & direction (controlled externally). */
  sortKey?: string;
  sortDir?: SortDir;
  onSort?: (key: string, dir: SortDir) => void;
}

// ---------------------------------------------------------------------------
// Column visibility hook (persisted to localStorage)
// ---------------------------------------------------------------------------

function useColumnVisibility(tableKey: string, columns: ColumnDef<unknown>[]) {
  const storageKey = `dt-cols:${tableKey}`;
  const defaults = Object.fromEntries(
    columns.map(c => [c.key, c.defaultVisible !== false])
  );
  const [visibility, setVisibility] = useState<Record<string, boolean>>(() => {
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) return { ...defaults, ...JSON.parse(stored) };
    } catch { /* ignore */ }
    return defaults;
  });

  const toggle = useCallback((key: string) => {
    setVisibility(prev => {
      const next = { ...prev, [key]: !prev[key] };
      localStorage.setItem(storageKey, JSON.stringify(next));
      return next;
    });
  }, [storageKey]);

  return { visibility, toggle };
}

// ---------------------------------------------------------------------------
// Inline edit cell
// ---------------------------------------------------------------------------

interface EditCellProps {
  initialValue: string;
  onCommit: (v: string) => void;
  onCancel: () => void;
  renderEdit: (value: string, onChange: (v: string) => void) => React.ReactNode;
}

function EditCell({ initialValue, onCommit, onCancel, renderEdit }: EditCellProps) {
  const [value, setValue] = useState(initialValue);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handle = (e: KeyboardEvent) => {
      if (e.key === 'Enter') { e.preventDefault(); onCommit(value); }
      if (e.key === 'Escape') { onCancel(); }
    };
    document.addEventListener('keydown', handle);
    return () => document.removeEventListener('keydown', handle);
  }, [value, onCommit, onCancel]);

  // Commit on click outside
  useEffect(() => {
    const handle = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onCommit(value);
      }
    };
    setTimeout(() => document.addEventListener('mousedown', handle), 0);
    return () => document.removeEventListener('mousedown', handle);
  }, [value, onCommit]);

  return (
    <div ref={ref} className="flex items-center gap-1 min-w-0">
      {renderEdit(value, setValue)}
      <button
        type="button"
        onMouseDown={e => { e.preventDefault(); onCommit(value); }}
        className="flex-shrink-0 p-0.5 text-green-400 hover:text-green-300">
        <Check className="w-3.5 h-3.5" />
      </button>
      <button
        type="button"
        onMouseDown={e => { e.preventDefault(); onCancel(); }}
        className="flex-shrink-0 p-0.5 text-dark-400 hover:text-white">
        <X className="w-3.5 h-3.5" />
      </button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

const inputCls = 'w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500';

export function DataTable<T>({
  tableKey,
  columns,
  data,
  getRowId,
  searchValue,
  onSearchChange,
  searchPlaceholder = 'Suchen…',
  filterSlot,
  bulkActions,
  onRowClick,
  emptyMessage = 'Keine Einträge vorhanden.',
  isLoading,
  sortKey,
  sortDir,
  onSort,
}: DataTableProps<T>) {
  const { visibility, toggle } = useColumnVisibility(tableKey, columns as ColumnDef<unknown>[]);
  const [showColPicker, setShowColPicker] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [editCell, setEditCell] = useState<{ rowId: string; colKey: string } | null>(null);

  const visibleCols = columns.filter(c => visibility[c.key] !== false);
  const allIds = data.map(getRowId);
  const allSelected = allIds.length > 0 && allIds.every(id => selected.has(id));
  const someSelected = selected.size > 0;

  const toggleRow = (id: string) => {
    setSelected(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const toggleAll = () => {
    if (allSelected) setSelected(new Set());
    else setSelected(new Set(allIds));
  };

  const clearSelection = () => setSelected(new Set());

  const handleSort = (key: string) => {
    if (!onSort) return;
    if (sortKey === key) {
      onSort(key, sortDir === 'asc' ? 'desc' : 'asc');
    } else {
      onSort(key, 'asc');
    }
  };

  const SortIcon = ({ colKey }: { colKey: string }) => {
    if (sortKey !== colKey) return <ChevronsUpDown className="w-3.5 h-3.5 opacity-30" />;
    return sortDir === 'asc'
      ? <ChevronUp className="w-3.5 h-3.5 text-primary-400" />
      : <ChevronDown className="w-3.5 h-3.5 text-primary-400" />;
  };

  return (
    <div className="flex flex-col gap-3">
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2">
        {onSearchChange && (
          <div className="relative flex-1 min-w-[180px] max-w-xs">
            <input
              type="text"
              value={searchValue ?? ''}
              onChange={e => onSearchChange(e.target.value)}
              placeholder={searchPlaceholder}
              className={inputCls}
            />
            {searchValue && (
              <button
                type="button"
                onClick={() => onSearchChange('')}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-dark-500 hover:text-white">
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
        )}
        {filterSlot}
        {/* Column picker toggle */}
        <div className="relative ml-auto">
          <button
            type="button"
            onClick={() => setShowColPicker(p => !p)}
            className="flex items-center gap-1.5 px-2.5 py-2 bg-dark-800 border border-dark-700 text-dark-300 rounded-lg hover:bg-dark-700 hover:text-white text-xs">
            <Columns3 className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Spalten</span>
          </button>
          {showColPicker && (
            <div className="absolute right-0 top-full mt-1 z-30 bg-dark-800 border border-dark-700 rounded-lg shadow-xl p-3 min-w-[180px]">
              <p className="text-xs text-dark-400 mb-2 font-medium">Sichtbare Spalten</p>
              {columns.map(col => (
                <label key={col.key} className="flex items-center gap-2 py-1 cursor-pointer hover:text-white text-sm text-dark-300">
                  <input
                    type="checkbox"
                    checked={visibility[col.key] !== false}
                    onChange={() => toggle(col.key)}
                    className="w-3.5 h-3.5 accent-primary-500"
                  />
                  {col.header}
                </label>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Bulk action bar */}
      {someSelected && bulkActions && bulkActions.length > 0 && (
        <div className="flex items-center gap-3 px-3 py-2 bg-primary-500/10 border border-primary-500/30 rounded-lg text-sm">
          <span className="text-primary-300 font-medium">{selected.size} ausgewählt</span>
          <div className="flex items-center gap-2 ml-2">
            {bulkActions.map((action, i) => (
              <button
                key={i}
                type="button"
                onClick={() => {
                  const rows = data.filter(r => selected.has(getRowId(r)));
                  action.onClick(rows);
                }}
                className={`flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                  action.variant === 'danger'
                    ? 'bg-red-500/20 text-red-300 hover:bg-red-500/30'
                    : 'bg-dark-700 text-dark-200 hover:bg-dark-600'
                }`}>
                {action.icon}
                {action.label}
              </button>
            ))}
          </div>
          <button
            type="button"
            onClick={clearSelection}
            className="ml-auto text-dark-500 hover:text-white">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Table */}
      <div className="overflow-x-auto rounded-lg border border-dark-800">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-dark-800">
              {bulkActions && (
                <th className="w-9 px-3 py-3">
                  <input
                    type="checkbox"
                    checked={allSelected}
                    onChange={toggleAll}
                    className="w-3.5 h-3.5 accent-primary-500"
                  />
                </th>
              )}
              {visibleCols.map(col => (
                <th
                  key={col.key}
                  className={`px-3 py-3 text-left text-xs text-dark-400 font-medium whitespace-nowrap ${col.width ?? ''} ${col.sortable && onSort ? 'cursor-pointer hover:text-white select-none' : ''}`}
                  onClick={col.sortable && onSort ? () => handleSort(col.key) : undefined}>
                  <span className="flex items-center gap-1">
                    {col.header}
                    {col.sortable && onSort && <SortIcon colKey={col.key} />}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <tr key={i} className="border-b border-dark-800/50">
                  {bulkActions && <td className="px-3 py-3" />}
                  {visibleCols.map(col => (
                    <td key={col.key} className="px-3 py-3">
                      <div className="h-3.5 bg-dark-700 rounded animate-pulse" style={{ width: `${50 + Math.random() * 40}%` }} />
                    </td>
                  ))}
                </tr>
              ))
            ) : data.length === 0 ? (
              <tr>
                <td
                  colSpan={visibleCols.length + (bulkActions ? 1 : 0)}
                  className="px-4 py-8 text-center text-dark-500 text-sm">
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              data.map(row => {
                const rowId = getRowId(row);
                const isSelected = selected.has(rowId);
                return (
                  <tr
                    key={rowId}
                    onClick={onRowClick ? () => onRowClick(row) : undefined}
                    className={`border-b border-dark-800/50 transition-colors ${
                      isSelected ? 'bg-primary-500/10' : 'hover:bg-dark-800/30'
                    } ${onRowClick ? 'cursor-pointer' : ''}`}>
                    {bulkActions && (
                      <td className="px-3 py-3" onClick={e => { e.stopPropagation(); toggleRow(rowId); }}>
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => toggleRow(rowId)}
                          className="w-3.5 h-3.5 accent-primary-500"
                        />
                      </td>
                    )}
                    {visibleCols.map(col => {
                      const isEditing = editCell?.rowId === rowId && editCell?.colKey === col.key;
                      const canEdit = !!col.renderEdit && !!col.onEdit;
                      return (
                        <td
                          key={col.key}
                          className={`px-3 py-2.5 text-dark-200 align-middle ${col.width ?? ''} ${canEdit && !isEditing ? 'group cursor-text' : ''}`}
                          onClick={canEdit && !isEditing ? (e) => { e.stopPropagation(); setEditCell({ rowId, colKey: col.key }); } : undefined}>
                          {isEditing && col.renderEdit && col.onEdit ? (
                            <EditCell
                              initialValue={col.getValue ? col.getValue(row) : ''}
                              onCommit={v => { col.onEdit!(row, v); setEditCell(null); }}
                              onCancel={() => setEditCell(null)}
                              renderEdit={(val, onChange) => col.renderEdit!(row, val, onChange)}
                            />
                          ) : (
                            <span className={canEdit ? 'group-hover:underline group-hover:decoration-dotted group-hover:decoration-dark-600' : ''}>
                              {col.render(row)}
                            </span>
                          )}
                        </td>
                      );
                    })}
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Convenience: default text edit input for renderEdit
// ---------------------------------------------------------------------------

export function textEditInput(placeholder?: string) {
  return (_row: unknown, value: string, onChange: (v: string) => void) => (
    <input
      autoFocus
      type="text"
      value={value}
      onChange={e => onChange(e.target.value)}
      placeholder={placeholder}
      className="w-full px-2 py-1 bg-dark-700 border border-primary-500 rounded text-white text-sm focus:outline-none min-w-0"
    />
  );
}

export function numberEditInput(placeholder?: string) {
  return (_row: unknown, value: string, onChange: (v: string) => void) => (
    <input
      autoFocus
      type="number"
      value={value}
      onChange={e => onChange(e.target.value)}
      placeholder={placeholder}
      className="w-full px-2 py-1 bg-dark-700 border border-primary-500 rounded text-white text-sm focus:outline-none min-w-0"
    />
  );
}

// Re-export Trash2 for convenience in bulk action icons
export { Trash2, SlidersHorizontal };
