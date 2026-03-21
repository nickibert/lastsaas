import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight, CheckSquare, Ship } from 'lucide-react';
import { Link } from 'react-router-dom';
import { calendarApi, type CalendarTask, type CalendarArrival } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';

function fmt(date: Date) {
  return date.toISOString().slice(0, 10);
}

function daysInMonth(year: number, month: number) {
  return new Date(year, month + 1, 0).getDate();
}

function firstDayOfMonth(year: number, month: number) {
  // 0=Sun → convert to Mon-first (0=Mon … 6=Sun)
  const d = new Date(year, month, 1).getDay();
  return (d + 6) % 7;
}

const MONTHS_DE = ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'];
const DAYS_DE = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So'];

interface DayEvents {
  tasks: CalendarTask[];
  arrivals: CalendarArrival[];
}

export default function CalendarPage() {
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;

  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth()); // 0-based

  const fromDate = new Date(year, month, 1);
  const toDate = new Date(year, month + 1, 0);

  const { data, isLoading } = useQuery({
    queryKey: ['calendar', year, month],
    queryFn: () => calendarApi.get(fmt(fromDate), fmt(toDate)),
    enabled,
    throwOnError: false,
  });

  const tasks = data?.tasks ?? [];
  const arrivals = data?.arrivals ?? [];

  // Build a map: dateStr → DayEvents
  const eventMap: Record<string, DayEvents> = {};
  const ensure = (d: string) => { if (!eventMap[d]) eventMap[d] = { tasks: [], arrivals: [] }; };

  tasks.forEach(t => { ensure(t.dueDate); eventMap[t.dueDate].tasks.push(t); });
  arrivals.forEach(a => {
    const d = a.actualDate || a.estimatedDate;
    if (d) { ensure(d); eventMap[d].arrivals.push(a); }
  });

  const prevMonth = () => {
    if (month === 0) { setMonth(11); setYear(y => y - 1); }
    else setMonth(m => m - 1);
  };
  const nextMonth = () => {
    if (month === 11) { setMonth(0); setYear(y => y + 1); }
    else setMonth(m => m + 1);
  };

  const days = daysInMonth(year, month);
  const firstDay = firstDayOfMonth(year, month); // 0=Mon
  const todayStr = fmt(now);

  // Build calendar grid (42 cells = 6 rows × 7 cols)
  const cells: (number | null)[] = [];
  for (let i = 0; i < firstDay; i++) cells.push(null);
  for (let d = 1; d <= days; d++) cells.push(d);
  while (cells.length % 7 !== 0) cells.push(null);

  const openTasks = tasks.filter(t => !t.doneAt).length;
  const doneTasks = tasks.filter(t => t.doneAt).length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Kalender</h1>
          <p className="text-dark-400 mt-1">
            {tasks.length} Aufgaben ({openTasks} offen, {doneTasks} erledigt) · {arrivals.length} Warenankünfte
          </p>
        </div>
        <div className="flex items-center gap-3">
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
      <div className="flex items-center gap-4 text-xs text-dark-400">
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-amber-500/30 border border-amber-500/50 inline-block" />
          Offene Aufgabe
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-emerald-500/30 border border-emerald-500/50 inline-block" />
          Erledigte Aufgabe
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-primary-500/30 border border-primary-500/50 inline-block" />
          Warenankunft (geplant)
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-teal-500/30 border border-teal-500/50 inline-block" />
          Warenankunft (eingetroffen)
        </span>
      </div>

      {isLoading ? (
        <div className="text-dark-400">Lädt...</div>
      ) : (
        <div className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
          {/* Weekday header */}
          <div className="grid grid-cols-7 border-b border-dark-800">
            {DAYS_DE.map(d => (
              <div key={d} className={`py-2 text-center text-xs font-medium ${d === 'Sa' || d === 'So' ? 'text-dark-500' : 'text-dark-400'}`}>
                {d}
              </div>
            ))}
          </div>

          {/* Calendar grid */}
          <div className="grid grid-cols-7">
            {cells.map((day, i) => {
              if (!day) return (
                <div key={`empty-${i}`} className="min-h-[100px] border-b border-r border-dark-800/50 bg-dark-950/30" />
              );
              const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
              const events = eventMap[dateStr];
              const isToday = dateStr === todayStr;
              const isWeekend = (i % 7) >= 5;

              return (
                <div key={dateStr}
                  className={`min-h-[100px] border-b border-r border-dark-800/50 p-1.5 ${isWeekend ? 'bg-dark-950/20' : ''}`}>
                  <div className={`text-xs font-medium mb-1 w-6 h-6 flex items-center justify-center rounded-full
                    ${isToday ? 'bg-primary-500 text-white' : isWeekend ? 'text-dark-500' : 'text-dark-400'}`}>
                    {day}
                  </div>
                  {events && (
                    <div className="space-y-0.5">
                      {events.tasks.map(t => (
                        <Link key={t.id} to={`/procurement/orders/${t.orderId}`}
                          className={`block text-xs px-1.5 py-0.5 rounded truncate leading-tight
                            ${t.doneAt
                              ? 'bg-emerald-500/20 text-emerald-400 line-through'
                              : 'bg-amber-500/20 text-amber-300'
                            }`}
                          title={`${t.orderNumber}: ${t.text}`}>
                          <CheckSquare className="w-2.5 h-2.5 inline mr-0.5" />
                          {t.text}
                        </Link>
                      ))}
                      {events.arrivals.map(a => (
                        <Link key={a.orderId} to={`/procurement/orders/${a.orderId}`}
                          className={`block text-xs px-1.5 py-0.5 rounded truncate leading-tight
                            ${a.actualDate
                              ? 'bg-teal-500/20 text-teal-400'
                              : 'bg-primary-500/20 text-primary-300'
                            }`}
                          title={`Bestellung ${a.orderNumber}${a.supplierName ? ' – ' + a.supplierName : ''}`}>
                          <Ship className="w-2.5 h-2.5 inline mr-0.5" />
                          {a.orderNumber}{a.supplierName ? ` – ${a.supplierName}` : ''}
                        </Link>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Detailed list below */}
      {(tasks.length > 0 || arrivals.length > 0) && (
        <div className="grid md:grid-cols-2 gap-6">
          {tasks.length > 0 && (
            <div className="bg-dark-900/50 border border-dark-800 rounded-xl p-4 space-y-2">
              <h3 className="text-sm font-semibold text-white mb-3">Aufgaben im Monat</h3>
              {tasks.map(t => (
                <div key={t.id} className="flex items-start gap-2 text-sm">
                  <span className={`mt-0.5 flex-shrink-0 ${t.doneAt ? 'text-emerald-400' : 'text-amber-400'}`}>
                    <CheckSquare className="w-4 h-4" />
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className={`${t.doneAt ? 'line-through text-dark-500' : 'text-white'} truncate`}>{t.text}</div>
                    <div className="text-dark-400 text-xs">
                      <Link to={`/procurement/orders/${t.orderId}`} className="hover:text-primary-400">
                        {t.orderNumber}
                      </Link>
                      {' · '}{new Date(t.dueDate + 'T00:00:00').toLocaleDateString('de-DE')}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {arrivals.length > 0 && (
            <div className="bg-dark-900/50 border border-dark-800 rounded-xl p-4 space-y-2">
              <h3 className="text-sm font-semibold text-white mb-3">Warenankünfte im Monat</h3>
              {arrivals.map(a => (
                <div key={a.orderId} className="flex items-start gap-2 text-sm">
                  <span className={`mt-0.5 flex-shrink-0 ${a.actualDate ? 'text-teal-400' : 'text-primary-400'}`}>
                    <Ship className="w-4 h-4" />
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="text-white">
                      <Link to={`/procurement/orders/${a.orderId}`} className="hover:text-primary-400">
                        {a.orderNumber}
                      </Link>
                      {a.supplierName && <span className="text-dark-400"> – {a.supplierName}</span>}
                    </div>
                    <div className="text-dark-400 text-xs">
                      {a.actualDate
                        ? `Eingetroffen: ${new Date(a.actualDate + 'T00:00:00').toLocaleDateString('de-DE')}`
                        : `Geplant: ${new Date(a.estimatedDate + 'T00:00:00').toLocaleDateString('de-DE')}`}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
