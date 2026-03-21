import { Link, useNavigate } from 'react-router-dom';
import { ShoppingCart, Package, Users, Ship, Anchor, Box, Globe, Tag, Gift } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { ordersApi, productsApi, suppliersApi } from '../../../api/procurement';

export default function ProcurementPage() {
  const { data: orders = [] } = useQuery({ queryKey: ['orders'], queryFn: () => ordersApi.list() });
  const { data: products = [] } = useQuery({ queryKey: ['products'], queryFn: () => productsApi.list() });
  const { data: suppliers = [] } = useQuery({ queryKey: ['suppliers'], queryFn: suppliersApi.list });

  const tiles = [
    {
      icon: ShoppingCart,
      label: 'Bestellungen',
      count: orders.length,
      to: '/procurement/orders',
      color: 'text-primary-400',
      bg: 'bg-primary-500/20',
    },
    {
      icon: Package,
      label: 'Produkte',
      count: products.length,
      to: '/procurement/products',
      color: 'text-accent-cyan',
      bg: 'bg-accent-cyan/20',
    },
    {
      icon: Users,
      label: 'Lieferanten',
      count: suppliers.length,
      to: '/procurement/suppliers',
      color: 'text-accent-purple',
      bg: 'bg-accent-purple/20',
    },
    {
      icon: Ship,
      label: 'Frachtführer',
      count: null,
      to: '/procurement/freight-carriers',
      color: 'text-amber-400',
      bg: 'bg-amber-500/20',
    },
    {
      icon: Anchor,
      label: 'Häfen',
      count: null,
      to: '/procurement/harbours',
      color: 'text-teal-400',
      bg: 'bg-teal-500/20',
    },
    {
      icon: Box,
      label: 'Container',
      count: null,
      to: '/procurement/containers',
      color: 'text-orange-400',
      bg: 'bg-orange-500/20',
    },
    {
      icon: Globe,
      label: 'Länder',
      count: null,
      to: '/procurement/countries',
      color: 'text-green-400',
      bg: 'bg-green-500/20',
    },
    {
      icon: Tag,
      label: 'Warengruppen',
      count: null,
      to: '/procurement/goods-groups',
      color: 'text-pink-400',
      bg: 'bg-pink-500/20',
    },
    {
      icon: Gift,
      label: 'Angebote',
      count: null,
      to: '/procurement/offers',
      color: 'text-indigo-400',
      bg: 'bg-indigo-500/20',
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
