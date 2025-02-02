'use client'

import { useState, useEffect } from 'react'
import { createClient } from '@/lib/supabase'

export default function CreateTunnel() {
  const [type, setType] = useState<'sit' | 'gre'>('sit')
  const [clientIPv4, setClientIPv4] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [serverIPv4, setServerIPv4] = useState<string | null>(null)

  useEffect(() => {
    // Pobierz publiczny IP serwera
    fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/system/status`, {
      headers: {
        'X-API-Key': process.env.NEXT_PUBLIC_API_KEY || '',
      },
    })
      .then(response => response.json())
      .then(data => {
        if (data.server_ipv4) {
          setServerIPv4(data.server_ipv4)
        }
      })
      .catch(err => {
        console.error('Error fetching server IP:', err)
        setError('Nie można pobrać adresu IP serwera')
      })
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)

    if (!serverIPv4) {
      setError('Nie można utworzyć tunelu - brak adresu IP serwera')
      setLoading(false)
      return
    }

    try {
      const supabase = createClient()
      const { data: { user } } = await supabase.auth.getUser()
      if (!user) throw new Error('Nie jesteś zalogowany')

      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/tunnels`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': process.env.NEXT_PUBLIC_API_KEY || '',
        },
        body: JSON.stringify({
          type,
          client_ipv4: clientIPv4,
          server_ipv4: serverIPv4,
          user_id: user.id,
        }),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || 'Błąd podczas tworzenia tunelu')
      }

      window.location.href = '/dashboard'
    } catch (error: any) {
      console.error('Error creating tunnel:', error)
      setError(error.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-100">
      <nav className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <a href="/dashboard" className="text-xl font-semibold text-gray-900 hover:text-gray-700">
                TunnelBroker
              </a>
            </div>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="max-w-3xl mx-auto">
            <div className="bg-white shadow sm:rounded-lg">
              <div className="px-4 py-5 sm:p-6">
                <h3 className="text-lg leading-6 font-medium text-gray-900">
                  Utwórz nowy tunel
                </h3>
                <div className="mt-2 max-w-xl text-sm text-gray-500">
                  <p>
                    Wybierz typ tunelu i podaj swój adres IPv4.
                  </p>
                </div>

                <form onSubmit={handleSubmit} className="mt-5 space-y-6">
                  {error && (
                    <div className="bg-red-50 text-red-500 p-4 rounded-md">
                      {error}
                    </div>
                  )}

                  <div>
                    <label className="block text-sm font-medium text-gray-700">
                      Typ tunelu
                    </label>
                    <select
                      value={type}
                      onChange={(e) => setType(e.target.value as 'sit' | 'gre')}
                      className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-primary-500 focus:border-primary-500 sm:text-sm rounded-md"
                    >
                      <option value="sit">SIT (Simple Internet Transition)</option>
                      <option value="gre">GRE (Generic Routing Encapsulation)</option>
                    </select>
                  </div>

                  <div>
                    <label htmlFor="clientIPv4" className="block text-sm font-medium text-gray-700">
                      Twój adres IPv4
                    </label>
                    <input
                      type="text"
                      id="clientIPv4"
                      value={clientIPv4}
                      onChange={(e) => setClientIPv4(e.target.value)}
                      placeholder="np. 192.0.2.1"
                      className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                      required
                    />
                    <p className="mt-2 text-sm text-gray-500">
                      Podaj publiczny adres IPv4, który będzie używany do zestawienia tunelu.
                    </p>
                  </div>

                  <div className="flex justify-end space-x-3">
                    <a
                      href="/dashboard"
                      className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
                    >
                      Anuluj
                    </a>
                    <button
                      type="submit"
                      disabled={loading}
                      className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
                    >
                      {loading ? 'Tworzenie...' : 'Utwórz tunel'}
                    </button>
                  </div>
                </form>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
} 