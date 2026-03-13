import { useState } from 'react';
import type { AdminReservationRequest } from '../../types/reservation';

export default function ReservationCreateForm() {
  const [form, setForm] = useState<AdminReservationRequest>({
    source: 'instagram',
    name: '',
    people: 2,
    visit_date: '',
    visit_time: '12:00',
    phone: '',
    email: '',
    note: '',
    status: 'approved',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setForm((prev) => ({
      ...prev,
      [name]: name === 'people' ? parseInt(value) : value,
    }));
    setErrors((prev) => {
      const next = { ...prev };
      delete next[name];
      return next;
    });
  };

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};
    if (!form.name.trim()) newErrors.name = 'お名前を入力してください';
    if (!form.visit_date) newErrors.visit_date = '来店日を入力してください';
    if (!form.visit_time) newErrors.visit_time = '来店時間を入力してください';
    if (!form.phone.trim()) newErrors.phone = '電話番号を入力してください';
    if (!form.email.trim()) newErrors.email = 'メールアドレスを入力してください';

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    setIsSubmitting(true);

    try {
      // TODO: POST /api/v1/admin/reservations に置き換え
      console.log('管理者予約登録:', form);
      await new Promise((resolve) => setTimeout(resolve, 500));
      setSuccess(true);
    } catch {
      setErrors({ submit: '登録に失敗しました' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const inputClass = (fieldName: string) =>
    `w-full px-4 py-3 rounded-lg border ${
      errors[fieldName] ? 'border-red-400 bg-red-50/50' : 'border-gray-200 bg-white'
    } focus:outline-none focus:ring-2 focus:ring-[#43676B]/30 focus:border-[#43676B] transition-colors text-base`;

  if (success) {
    return (
      <div className="text-center py-12 space-y-4">
        <div className="w-16 h-16 rounded-full bg-green-100 flex items-center justify-center mx-auto">
          <svg className="w-8 h-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
        </div>
        <p className="text-lg font-medium">予約を登録しました</p>
        <div className="flex gap-3 justify-center">
          <a
            href="/admin/reservations"
            className="px-4 py-2 bg-[#43676B] text-white rounded-lg hover:bg-[#365558] transition-colors text-sm"
          >
            予約一覧へ
          </a>
          <button
            onClick={() => {
              setSuccess(false);
              setForm({
                source: 'instagram',
                name: '',
                people: 2,
                visit_date: '',
                visit_time: '12:00',
                phone: '',
                email: '',
                note: '',
                status: 'approved',
              });
            }}
            className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors text-sm"
          >
            続けて登録
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-medium">予約登録</h1>

      <form onSubmit={handleSubmit} className="max-w-lg space-y-5">
        {/* 予約経路 */}
        <div>
          <label className="block text-sm font-medium mb-2">
            予約経路 <span className="text-red-500">*</span>
          </label>
          <select name="source" value={form.source} onChange={handleChange} className={inputClass('source')}>
            <option value="instagram">Instagram</option>
            <option value="phone">電話</option>
            <option value="walk_in">ウォークイン</option>
            <option value="other">その他</option>
          </select>
        </div>

        {/* 来店日 */}
        <div>
          <label className="block text-sm font-medium mb-2">
            来店日 <span className="text-red-500">*</span>
          </label>
          <input type="date" name="visit_date" value={form.visit_date} onChange={handleChange} className={inputClass('visit_date')} />
          {errors.visit_date && <p className="text-red-500 text-sm mt-1">{errors.visit_date}</p>}
        </div>

        {/* 来店時間 */}
        <div>
          <label className="block text-sm font-medium mb-2">
            来店時間 <span className="text-red-500">*</span>
          </label>
          <select name="visit_time" value={form.visit_time} onChange={handleChange} className={inputClass('visit_time')}>
            {['8:30', '9:00', '9:30', '10:00', '10:30', '11:00', '11:30', '12:00', '12:30', '13:00', '13:30', '14:00'].map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
          {errors.visit_time && <p className="text-red-500 text-sm mt-1">{errors.visit_time}</p>}
        </div>

        {/* 人数 */}
        <div>
          <label className="block text-sm font-medium mb-2">
            人数 <span className="text-red-500">*</span>
          </label>
          <select name="people" value={form.people} onChange={handleChange} className={inputClass('people')}>
            {Array.from({ length: 10 }, (_, i) => i + 1).map((n) => (
              <option key={n} value={n}>{n}名</option>
            ))}
          </select>
        </div>

        {/* お名前 */}
        <div>
          <label className="block text-sm font-medium mb-2">
            お名前 <span className="text-red-500">*</span>
          </label>
          <input type="text" name="name" value={form.name} onChange={handleChange} placeholder="山田太郎" className={inputClass('name')} />
          {errors.name && <p className="text-red-500 text-sm mt-1">{errors.name}</p>}
        </div>

        {/* 電話番号 */}
        <div>
          <label className="block text-sm font-medium mb-2">
            電話番号 <span className="text-red-500">*</span>
          </label>
          <input type="tel" name="phone" value={form.phone} onChange={handleChange} placeholder="090-1234-5678" className={inputClass('phone')} />
          {errors.phone && <p className="text-red-500 text-sm mt-1">{errors.phone}</p>}
        </div>

        {/* メールアドレス */}
        <div>
          <label className="block text-sm font-medium mb-2">
            メールアドレス <span className="text-red-500">*</span>
          </label>
          <input type="email" name="email" value={form.email} onChange={handleChange} placeholder="yamada@example.com" className={inputClass('email')} />
          {errors.email && <p className="text-red-500 text-sm mt-1">{errors.email}</p>}
        </div>

        {/* 備考 */}
        <div>
          <label className="block text-sm font-medium mb-2">備考</label>
          <textarea name="note" value={form.note} onChange={handleChange} rows={3} className={inputClass('note')} />
        </div>

        {/* ステータス */}
        <div>
          <label className="block text-sm font-medium mb-2">ステータス</label>
          <select name="status" value={form.status} onChange={handleChange} className={inputClass('status')}>
            <option value="approved">承認済み</option>
            <option value="pending">申請中</option>
          </select>
        </div>

        {errors.submit && (
          <div className="rounded-lg bg-red-50 border border-red-200 p-3 text-red-700 text-sm">
            {errors.submit}
          </div>
        )}

        <button
          type="submit"
          disabled={isSubmitting}
          className={`w-full py-3 rounded-xl text-white font-medium transition-all ${
            isSubmitting
              ? 'bg-gray-400 cursor-not-allowed'
              : 'bg-[#43676B] hover:bg-[#365558] active:scale-[0.98] shadow-lg'
          }`}
        >
          {isSubmitting ? '登録中...' : '登録する'}
        </button>
      </form>
    </div>
  );
}
