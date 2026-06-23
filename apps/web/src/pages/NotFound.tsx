import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export default function NotFound() {
  const { t } = useTranslation()
  return (
    <div className="mx-auto max-w-content px-4 lg:px-8 py-24 text-center">
      <p className="mono text-2xs text-fg-muted">HTTP 404</p>
      <h1 className="text-3xl font-bold mt-3 mb-3">{t('errors.notFound.title')}</h1>
      <p className="text-fg-subtle prose-measure mx-auto mb-6">{t('errors.notFound.body')}</p>
      <Link to="/" className="text-accent underline">Go home →</Link>
    </div>
  )
}
