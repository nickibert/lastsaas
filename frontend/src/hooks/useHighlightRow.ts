import { useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';

/**
 * Reads ?highlight=<id> from the URL and scrolls to + briefly flashes
 * the table row with data-highlight-id="<id>" once the list has loaded.
 *
 * Usage in a list page:
 *   useHighlightRow(items.length > 0);
 *
 * Usage on each row:
 *   <tr data-highlight-id={item.id} ...>
 */
export function useHighlightRow(ready: boolean) {
  const [searchParams] = useSearchParams();
  const id = searchParams.get('highlight');

  useEffect(() => {
    if (!ready || !id) return;
    const el = document.querySelector<HTMLElement>(`[data-highlight-id="${id}"]`);
    if (!el) return;
    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    el.classList.add('highlight-row');
    const timer = setTimeout(() => el.classList.remove('highlight-row'), 2000);
    return () => clearTimeout(timer);
  }, [ready, id]);
}
