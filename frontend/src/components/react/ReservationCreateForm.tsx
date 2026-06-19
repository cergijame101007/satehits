import { useState, type FormEvent } from 'react';
import type { AdminReservationRequest } from '../../types/reservation';
import { createAdminReservation, ReservationApiError } from '@/lib/adminReservation';
import { getAvailability } from '@/lib/availability';
import DatePickerField from '@/components/react/DatePickerField';
import Alert from '@/components/react/ui/Alert';
import Button from '@/components/react/ui/Button';
import ButtonLink from '@/components/react/ui/ButtonLink';
import Textarea from '@/components/react/ui/Textarea';
import { inputClassName } from '@/lib/ui/inputStyles';

/** pending / approved は reserved 集計対象。超過時は登録前に確認する */
async function confirmIfCapacityExceeded(
  visitDate: string,
  people: number,
  status: string,
): Promise<boolean> {
  if (status !== 'pending' && status !== 'approved') {
    return true;
  }

  try {
    const availability = await getAvailability(visitDate);
    if (availability.is_holiday) {
      return true;
    }

    const { capacity, reserved } = availability;
    if (reserved + people <= capacity) {
      return true;
    }

    const total = reserved + people;
    return window.confirm(
      `提供可能数（${capacity}食）を超えて登録されます（予約済み ${reserved}食 + 今回 ${people}食 = ${total}食）。このまま登録しますか？`,
    );
  } catch {
    return true;
  }
}

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
      delete next.submit;
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

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!validate()) return;

    const status = form.status ?? 'approved';
    const confirmed = await confirmIfCapacityExceeded(form.visit_date, form.people, status);
    if (!confirmed) return;

    setIsSubmitting(true);
    setErrors({});

    try {
      await createAdminReservation(form);
      setSuccess(true);
    } catch (err) {
      if (err instanceof ReservationApiError) {
        if (err.details?.length) {
          const fieldErrors: Record<string, string> = {};
          for (const detail of err.details) {
            fieldErrors[detail.field] = detail.message;
          }
          setErrors(fieldErrors);
        } else {
          setErrors({ submit: err.message });
        }
      } else {
        setErrors({ submit: '登録に失敗しました' });
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const inputClass = (fieldName: string) => inputClassName(!!errors[fieldName]);

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
          <ButtonLink href="/admin/reservations" variant="primary" size="md">
            予約一覧へ
          </ButtonLink>
          <Button
            variant="ghost"
            size="md"
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
          >
            続けて登録
          </Button>
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
          {errors.source && <p className="text-red-500 text-sm mt-1">{errors.source}</p>}
        </div>

        {/* 来店日 */}
        <DatePickerField
          value={form.visit_date}
          onChange={(date) => {
            setForm((prev) => ({ ...prev, visit_date: date }));
            setErrors((prev) => {
              const next = { ...prev };
              delete next.visit_date;
              delete next.submit;
              return next;
            });
          }}
          error={errors.visit_date}
          required
        />

        {/* 来店時間 */}
        <div>
          <label className="block text-sm font-medium mb-2">
            来店時間 <span className="text-red-500">*</span>
          </label>
          <select name="visit_time" value={form.visit_time} onChange={handleChange} className={inputClass('visit_time')}>
            {['8:30', '9:00', '9:30', '10:00', '10:30', '11:00', '11:30', '12:00', '12:30', '13:00', '13:30'].map((t) => (
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
            {Array.from({ length: 7 }, (_, i) => i + 1).map((n) => (
              <option key={n} value={n}>{n}名</option>
            ))}
          </select>
          {errors.people && <p className="text-red-500 text-sm mt-1">{errors.people}</p>}
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
          <Textarea name="note" value={form.note} onChange={handleChange} rows={3} hasError={!!errors.note} />
        </div>

        {/* ステータス */}
        <div>
          <label className="block text-sm font-medium mb-2">ステータス</label>
          <select name="status" value={form.status ?? 'approved'} onChange={handleChange} className={inputClass('status')}>
            <option value="approved">承認済み</option>
            <option value="pending">申請中</option>
          </select>
        </div>

        {errors.submit && <Alert>{errors.submit}</Alert>}

        <Button type="submit" variant="primary" size="lg" disabled={isSubmitting}>
          {isSubmitting ? '登録中...' : '登録する'}
        </Button>
      </form>
    </div>
  );
}
