import { useState, type FormEvent } from 'react';
import { AuthError, login } from '@/lib/auth';
import Alert from '@/components/react/ui/Alert';
import Button from '@/components/react/ui/Button';
import { inputClassName } from '@/components/react/ui/inputStyles';

export default function LoginForm() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError('');

    if (!email.trim() || !password.trim()) {
      setError('メールアドレスとパスワードを入力してください');
      return;
    }

    setIsSubmitting(true);

    try {
      await login(email, password);
      window.location.href = '/admin';
    } catch (err) {
      if (err instanceof AuthError) {
        if (err.code === 'UNAUTHORIZED') {
          setError('メールアドレスまたはパスワードが正しくありません');
        } else if (err.code === 'VALIDATION_ERROR') {
          setError(err.message);
        } else {
          setError('ログインに失敗しました。時間をおいて再度お試しください。');
        }
      } else {
        setError('ログインに失敗しました。時間をおいて再度お試しください。');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-medium text-center mb-8">管理者ログイン</h1>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-sm font-medium mb-2">メールアドレス</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={inputClassName()}
              autoComplete="email"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">パスワード</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={inputClassName()}
              autoComplete="current-password"
              required
            />
          </div>

          {error && <Alert variant="error">{error}</Alert>}

          <Button type="submit" variant="primary" size="lg" disabled={isSubmitting}>
            {isSubmitting ? 'ログイン中...' : 'ログイン'}
          </Button>
        </form>
      </div>
    </div>
  );
}
