'use client'

import { useEffect, useState } from 'react'
import { createClient } from '@/lib/supabase'

interface Tunnel {
  id: string
  type: 'sit' | 'gre'
  client_ipv4: string
  server_ipv4: string
  status: string
  created_at: string
}

export default function Dashboard() {
  const [tunnels, setTunnels] = useState<Tunnel[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [user, setUser] = useState<any>(null)

  useEffect(() => {
    const fetchUser = async () => {
      const supabase = createClient()
      const { data: { user } } = await supabase.auth.getUser()
      setUser(user)
    }

    const fetchTunnels = async () => {
      try {
        const response = await fetch('http://localhost:9090/api/v1/tunnels', {
          headers: {
            'X-API-Key': 'frontend-key',
          },
        })
        if (!response.ok) throw new Error('Błąd podczas pobierania tuneli')
        const data = await response.json()
        setTunnels(data)
      } catch (error: any) {
        setError(error.message)
      } finally {
        setLoading(false)
      }
    }

    fetchUser()
    fetchTunnels()
  }, [])

  const handleSignOut = async () => {
    const supabase = createClient()
    await supabase.auth.signOut()
    window.location.href = '/'
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-lg">Ładowanie...</div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-100">
      <nav className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <h1 className="text-xl font-semibold">TunnelBroker</h1>
            </div>
            <div className="flex items-center space-x-4">
              <span className="text-gray-700">{user?.email}</span>
              <button
                onClick={handleSignOut}
                className="px-3 py-2 text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-100 rounded-md"
              >
                Wyloguj
              </button>
            </div>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="flex justify-between items-center mb-6">
            <h2 className="text-2xl font-bold text-gray-900">Twoje tunele</h2>
            {tunnels.length < 2 && (
              <a
                href="/dashboard/tunnels/create"
                className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
              >
                Utwórz nowy tunel
              </a>
            )}
          </div>

          {error && (
            <div className="bg-red-50 text-red-500 p-4 rounded-md mb-6">
              {error}
            </div>
          )}

          <div className="bg-white shadow overflow-hidden sm:rounded-lg">
            <ul className="divide-y divide-gray-200">
              {tunnels.map((tunnel) => (
                <li key={tunnel.id}>
                  <a href={`/dashboard/tunnels/${tunnel.id}`} className="block hover:bg-gray-50">
                    <div className="px-4 py-4 sm:px-6">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-3">
                          <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                            tunnel.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                          }`}>
                            {tunnel.status}
                          </span>
                          <p className="text-sm font-medium text-primary-600 truncate">
                            {tunnel.type.toUpperCase()} - {tunnel.client_ipv4}
                          </p>
                        </div>
                        <div className="text-sm text-gray-500">
                          Utworzono: {new Date(tunnel.created_at).toLocaleDateString()}
                        </div>
                      </div>
                    </div>
                  </a>
                </li>
              ))}
              {tunnels.length === 0 && (
                <li className="px-4 py-8 text-center text-gray-500">
                  Nie masz jeszcze żadnych tuneli. Utwórz swój pierwszy tunel!
                </li>
              )}
            </ul>
          </div>
        </div>
      </main>
    </div>
  )
} 