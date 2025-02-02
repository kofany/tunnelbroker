'use client'

import { useEffect } from 'react'
import { createClient } from '@/lib/supabase'

export default function AuthCallback() {
  useEffect(() => {
    const handleCallback = async () => {
      const supabase = createClient()
      try {
        const { error } = await supabase.auth.getSession()
        if (error) throw error
        window.location.href = '/dashboard'
      } catch (error) {
        console.error('Błąd podczas logowania:', error)
        window.location.href = '/'
      }
    }

    handleCallback()
  }, [])

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <h2 className="text-2xl font-semibold text-gray-900">
          Trwa logowanie...
        </h2>
        <p className="mt-2 text-gray-600">
          Proszę czekać, za chwilę zostaniesz przekierowany.
        </p>
      </div>
    </div>
  )
} 