import { useState } from 'react'
import { Alert, KeyboardAvoidingView, Platform, Pressable, StyleSheet, Text, TextInput, View } from 'react-native'
import { router } from 'expo-router'
import { api, storeAuth } from '../lib/api'
import { COLORS } from '../lib/theme'
import { ApiException, type MeResponse } from '@alethea/shared-types'

interface LoginResp {
  user: MeResponse['user']
  sessionToken: string
  csrfToken: string
}

export default function Login() {
  const [email, setEmail] = useState('')
  const [pw, setPw] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit() {
    setBusy(true)
    try {
      const r = await api<LoginResp>('/auth/login', {
        method: 'POST',
        body: { email: email.trim(), password: pw, client: 'mobile' },
      })
      await storeAuth(r.sessionToken, r.csrfToken)
      router.replace('/')
    } catch (e) {
      Alert.alert('Sign-in failed', e instanceof ApiException ? e.message : 'Unknown error')
    } finally {
      setBusy(false)
    }
  }

  return (
    <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : undefined} style={styles.root}>
      <View style={styles.body}>
        <Text style={styles.h1}>Sign in</Text>
        <TextInput
          style={styles.input}
          value={email}
          onChangeText={setEmail}
          placeholder="email"
          placeholderTextColor={COLORS.fgSubtle}
          keyboardType="email-address"
          autoCapitalize="none"
          autoComplete="email"
          editable={!busy}
        />
        <TextInput
          style={styles.input}
          value={pw}
          onChangeText={setPw}
          placeholder="password"
          placeholderTextColor={COLORS.fgSubtle}
          secureTextEntry
          editable={!busy}
        />
        <Pressable
          disabled={busy || !email || !pw}
          onPress={submit}
          style={({ pressed }) => [
            styles.cta,
            (busy || !email || !pw) && styles.ctaDisabled,
            pressed && { opacity: 0.85 },
          ]}
        >
          <Text style={styles.ctaLabel}>{busy ? 'Signing in…' : 'Sign in'}</Text>
        </Pressable>
        <Text style={styles.note}>
          Mobile uses a session-token header instead of cookies. The web app continues to use HttpOnly cookies.
        </Text>
      </View>
    </KeyboardAvoidingView>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: COLORS.bg },
  body: { flex: 1, padding: 24, justifyContent: 'center' },
  h1: { color: COLORS.fg, fontSize: 28, fontWeight: '700', marginBottom: 24 },
  input: {
    marginTop: 12,
    height: 48,
    color: COLORS.fg,
    backgroundColor: COLORS.bgElevated,
    borderColor: COLORS.border,
    borderWidth: 1,
    borderRadius: 12,
    paddingHorizontal: 14,
    fontSize: 16,
  },
  cta: {
    marginTop: 24, height: 52, borderRadius: 12, backgroundColor: COLORS.accent,
    alignItems: 'center', justifyContent: 'center',
  },
  ctaDisabled: { opacity: 0.5 },
  ctaLabel: { color: COLORS.accentFg, fontSize: 16, fontWeight: '600' },
  note: { color: COLORS.fgSubtle, fontSize: 12, marginTop: 24, lineHeight: 18 },
})
