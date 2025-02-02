import { createServerClient, type CookieOptions } from '@supabase/ssr'
import { NextResponse, type NextRequest } from 'next/server'

export async function middleware(request: NextRequest) {
  let response = NextResponse.next({
    request: {
      headers: request.headers,
    },
  })

  const supabase = createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        get(name: string) {
          return request.cookies.get(name)?.value
        },
        set(name: string, value: string, options: CookieOptions) {
          response.cookies.set({
            name,
            value,
            ...options,
          })
        },
        remove(name: string, options: CookieOptions) {
          response.cookies.set({
            name,
            value: '',
            ...options,
          })
        },
      },
    }
  )

  const { data: { session } } = await supabase.auth.getSession()

  // Chronione ścieżki
  const protectedPaths = ['/dashboard']
  const isProtectedPath = protectedPaths.some(path => 
    request.nextUrl.pathname.startsWith(path)
  )

  // Ścieżki dla niezalogowanych
  const authPaths = ['/', '/register']
  const isAuthPath = authPaths.some(path => 
    request.nextUrl.pathname === path
  )

  // Przekierowanie zalogowanych użytkowników z auth paths do dashboard
  if (session && isAuthPath) {
    return NextResponse.redirect(new URL('/dashboard', request.url))
  }

  // Przekierowanie niezalogowanych użytkowników do strony logowania
  if (!session && isProtectedPath) {
    return NextResponse.redirect(new URL('/', request.url))
  }

  return response
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
}
