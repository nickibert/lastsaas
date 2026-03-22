import { Link } from 'react-router-dom';
import { ShoppingCart, Package, Users, Ship, Anchor, Box, Globe, Tag, Gift, CalendarDays, UserCheck, Warehouse, Settings } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { ordersApi, productsApi, suppliersApi, freightCarriersApi, harboursApi, containersApi, countriesApi, goodsGroupsApi, offersApi, customersApi, stockApi } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';

export default function ProcurementPage() {
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const { data: ordersData } = useQuery({ queryKey: ['orders', 1, 25], queryFn: () => ordersApi.list(), enabled, throwOnError: false });
  const { data: productsData } = useQuery({ queryKey: ['products'], queryFn: () => productsApi.list(), enabled, throwOnError: false });
  const { data: suppliersData } = useQuery({ queryKey: ['suppliers', 1, 25], queryFn: () => suppliersApi.list(), enabled, throwOnError: false });
  const { data: freightCarriersData } = useQuery({ queryKey: ['freight-carriers', 1, 1], queryFn: () => freightCarriersApi.list(1, 1), enabled, throwOnError: false });
  const { data: harboursData } = useQuery({ queryKey: ['harbours', 1, 1], queryFn: () => harboursApi.list(1, 1), enabled, throwOnError: false });
  const { data: containersData } = useQuery({ queryKey: ['containers', 1, 1], queryFn: () => containersApi.list(1, 1), enabled, throwOnError: false });
  const { data: countriesData } = useQuery({ queryKey: ['countries', 1, 1], queryFn: () => countriesApi.list(1, 1), enabled, throwOnError: false });
  const { data: goodsGroupsData } = useQuery({ queryKey: ['goods-groups', 1, 1], queryFn: () => goodsGroupsApi.list(1, 1), enabled, throwOnError: false });
  const { data: offersData } = useQuery({ queryKey: ['offers', 1, 1], queryFn: () => offersApi.list(undefined, 1, 1), enabled, throwOnError: false });
  const { data: customersData } = useQuery({ queryKey: ['customers', 1, 1], queryFn: () => customersApi.list(undefined, 1, 1), enabled, throwOnError: false });
  const { data: stockData } = useQuery({ queryKey: ['stock-levels', 1, 1], queryFn: () => stockApi.listLevels({ page: 1, limit: 1 }), enabled, throwOnError: false });

  const tiles = [
    {
      icon: ShoppingCart,
      label: 'Bestellungen',
      count: ordersData?.total ?? null,
      to: '/procurement/orders',
      color: 'text-primary-400',
      bg: 'bg-primary-500/20',
    },
    {
      icon: Package,
      label: 'Produkte',
      count: productsData?.total ?? null,
      to: '/procurement/products',
      color: 'text-accent-cyan',
      bg: 'bg-accent-cyan/20',
    },
    {
      icon: Users,
      label: 'Lieferanten',
      count: suppliersData?.total ?? null,
      to: '/procurement/suppliers',
      color: 'text-accent-purple',
      bg: 'bg-accent-purple/20',
    },
    {
      icon: Ship,
      label: 'Frachtführer',
      count: freightCarriersData?.total ?? null,
      to: '/procurement/freight-carriers',
      color: 'text-amber-400',
      bg: 'bg-amber-500/20',
    },
    {
      icon: Anchor,
      label: 'Häfen',
      count: harboursData?.total ?? null,
      to: '/procurement/harbours',
      color: 'text-teal-400',
      bg: 'bg-teal-500/20',
    },
    {
      icon: Box,
      label: 'Container',
      count: containersData?.total ?? null,
      to: '/procurement/containers',
      color: 'text-orange-400',
      bg: 'bg-orange-500/20',
    },
    {
      icon: Globe,
      label: 'Länder',
      count: countriesData?.total ?? null,
      to: '/procurement/countries',
      color: 'text-green-400',
      bg: 'bg-green-500/20',
    },
    {
      icon: Tag,
      label: 'Warengruppen',
      count: goodsGroupsData?.total ?? null,
      to: '/procurement/goods-groups',
      color: 'text-pink-400',
      bg: 'bg-pink-500/20',
    },
    {
      icon: Gift,
      label: 'Angebote',
      count: offersData?.total ?? null,
      to: '/procurement/offers',
      color: 'text-indigo-400',
      bg: 'bg-indigo-500/20',
    },
    {
      icon: CalendarDays,
      label: 'Kalender',
      count: null,
      to: '/procurement/calendar',
      color: 'text-violet-400',
      bg: 'bg-violet-500/20',
    },
    {
      icon: UserCheck,
      label: 'Kunden',
      count: customersData?.total ?? null,
      to: '/procurement/customers',
      color: 'text-cyan-400',
      bg: 'bg-cyan-500/20',
    },
    {
      icon: Warehouse,
      label: 'Lager',
      count: stockData?.total ?? null,
      to: '/procurement/inventory',
      color: 'text-lime-400',
      bg: 'bg-lime-500/20',
    },
    {
      icon: Settings,
      label: 'Konfiguration',
      count: null,
      to: '/procurement/config',
      color: 'text-dark-400',
      bg: 'bg-dark-700/50',
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white">Einkauf</h1>
        <p className="text-dark-400 mt-1">Beschaffung, Lieferanten und Bestellverwaltung</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {tiles.map(({ icon: Icon, label, count, to, color, bg }) => (
          <Link
            key={to}
            to={to}
            className="bg-dark-900/50 backdrop-blur-sm border border-dark-800 rounded-2xl p-6 hover:border-dark-700 transition-colors"
          >
            <div className={`w-12 h-12 rounded-xl ${bg} flex items-center justify-center mb-4`}>
              <Icon className={`w-6 h-6 ${color}`} />
            </div>
            <h3 className="text-lg font-semibold text-white mb-1">{label}</h3>
            {count !== null && (
              <p className="text-sm text-dark-400">{count} Einträge</p>
            )}
          </Link>
        ))}
      </div>
    </div>
  );
}
