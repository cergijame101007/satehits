import { useState, useEffect, useMemo, type FormEvent } from 'react';
import type { ReservationRequest, AvailabilityResponse } from '../../types/reservation';
import { fetchAvailabilityMapForRange, toAvailabilityErrorMessage } from '@/lib/availability';

/** 日付を YYYY-MM-DD 形式にフォーマット */
function formatDate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

/** 日付表示用フォーマット */
function formatDateJa(dateStr: string): string {
  const date = new Date(dateStr);
  const weekdays = ['日', '月', '火', '水', '木', '金', '土'];
  return `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日（${weekdays[date.getDay()]}）`;
}

/** 予約可能な日付範囲を取得 */
function getBookableRange(): { min: Date; max: Date } {
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  tomorrow.setHours(0, 0, 0, 0);
  const maxDate = new Date();
  maxDate.setDate(maxDate.getDate() + 14);
  maxDate.setHours(0, 0, 0, 0);
  return { min: tomorrow, max: maxDate };
}

/** 予約可能な来店時間の選択肢（11:30-13:30。日曜朝営業の朝 8:30〜は予約不可） */
function getTimeSlots(): string[] {
  const slots: string[] = [];
  let h = 11;
  let m = 30;

  while (h < 13 || (h === 13 && m <= 30)) {
    slots.push(`${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`);
    m += 30;
    if (m >= 60) {
      h++;
      m = 0;
    }
  }

  return slots;
}

// --- Calendar Component ---

interface CalendarProps {
  selectedDate: string | null;
  onSelect: (date: string) => void;
  availabilityMap: Map<string, AvailabilityResponse>;
}

function Calendar({ selectedDate, onSelect, availabilityMap }: CalendarProps) {
  const [viewDate, setViewDate] = useState(() => {
    const now = new Date();
    return { year: now.getFullYear(), month: now.getMonth() };
  });

  const { min: minDate, max: maxDate } = getBookableRange();

  const calendarDays = useMemo(() => {
    const firstDay = new Date(viewDate.year, viewDate.month, 1);
    const lastDay = new Date(viewDate.year, viewDate.month + 1, 0);
    const startPad = firstDay.getDay();

    const days: (Date | null)[] = [];
    for (let i = 0; i < startPad; i++) days.push(null);
    for (let d = 1; d <= lastDay.getDate(); d++) {
      days.push(new Date(viewDate.year, viewDate.month, d));
    }
    return days;
  }, [viewDate.year, viewDate.month]);

  const prevMonth = () => {
    setViewDate((prev) => {
      const m = prev.month - 1;
      return m < 0 ? { year: prev.year - 1, month: 11 } : { year: prev.year, month: m };
    });
  };

  const nextMonth = () => {
    setViewDate((prev) => {
      const m = prev.month + 1;
      return m > 11 ? { year: prev.year + 1, month: 0 } : { year: prev.year, month: m };
    });
  };

  const weekdays = ['日', '月', '火', '水', '木', '金', '土'];

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <button
          type="button"
          onClick={prevMonth}
          className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors"
        >
          ‹
        </button>
        <span className="text-lg font-medium">
          {viewDate.year}年{viewDate.month + 1}月
        </span>
        <button
          type="button"
          onClick={nextMonth}
          className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors"
        >
          ›
        </button>
      </div>

      {/* Weekday headers */}
      <div className="grid grid-cols-7 mb-1">
        {weekdays.map((w, i) => (
          <div
            key={w}
            className={`text-center text-xs font-medium py-1 ${
              i === 0 ? 'text-red-400' : i === 6 ? 'text-blue-400' : 'text-gray-500'
            }`}
          >
            {w}
          </div>
        ))}
      </div>

      {/* Days */}
      <div className="grid grid-cols-7 gap-1">
        {calendarDays.map((date, i) => {
          if (!date) return <div key={`pad-${i}`} />;

          const dateStr = formatDate(date);
          const isInRange = date >= minDate && date <= maxDate;
          const availability = availabilityMap.get(dateStr);
          const isHoliday = availability?.is_holiday === true;
          const isSelectable =
            isInRange && availability !== undefined && !isHoliday && availability.available > 0;
          const isSelected = selectedDate === dateStr;

          return (
            <button
              key={dateStr}
              type="button"
              disabled={!isSelectable}
              onClick={() => isSelectable && onSelect(dateStr)}
              className={`relative flex flex-col items-center py-1.5 rounded-lg text-sm transition-colors ${
                isSelected
                  ? 'bg-primary text-white'
                  : isSelectable
                    ? 'hover:bg-primary/10 cursor-pointer'
                    : 'text-gray-300 cursor-not-allowed'
              } ${isHoliday && !isSelected ? 'bg-gray-50' : ''}`}
            >
              <span>{date.getDate()}</span>
              {isInRange && !isHoliday && availability && (
                <span
                  className={`text-[10px] leading-none ${
                    isSelected ? 'text-white/80' : availability.available <= 3 ? 'text-red-500' : 'text-primary'
                  }`}
                >
                  残{availability.available}
                </span>
              )}
              {isHoliday && isInRange && (
                <span className={`text-[10px] leading-none ${isSelected ? 'text-white/80' : 'text-gray-400'}`}>
                  休
                </span>
              )}
            </button>
          );
        })}
      </div>

      <p className="text-xs text-gray-400 mt-3">
        ※ 翌日〜14日先まで予約可能です
      </p>
    </div>
  );
}

// --- Main ReservationForm Component ---

export default function ReservationForm() {
  const [selectedDate, setSelectedDate] = useState<string | null>(null);
  const [form, setForm] = useState({
    visit_time: '',
    people: '2',
    name: '',
    phone: '',
    email: '',
    note: '',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [availabilityMap, setAvailabilityMap] = useState<Map<string, AvailabilityResponse>>(new Map());
  const [isAvailabilityLoading, setIsAvailabilityLoading] = useState(true);
  const [availabilityError, setAvailabilityError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const { min, max } = getBookableRange();

    async function loadAvailability() {
      setIsAvailabilityLoading(true);
      setAvailabilityError(null);
      try {
        const map = await fetchAvailabilityMapForRange(min, max);
        if (!cancelled) {
          setAvailabilityMap(map);
        }
      } catch (err) {
        if (!cancelled) {
          setAvailabilityError(toAvailabilityErrorMessage(err));
          setAvailabilityMap(new Map());
          setSelectedDate(null);
        }
      } finally {
        if (!cancelled) {
          setIsAvailabilityLoading(false);
        }
      }
    }

    loadAvailability();
    return () => {
      cancelled = true;
    };
  }, []);

  const timeSlots = selectedDate ? getTimeSlots() : [];

  const handleDateSelect = (dateStr: string) => {
    setSelectedDate(dateStr);
    setForm((prev) => ({ ...prev, visit_time: '' }));
    setErrors((prev) => {
      const next = { ...prev };
      delete next.visit_date;
      return next;
    });
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setForm((prev) => ({ ...prev, [name]: value }));
    setErrors((prev) => {
      const next = { ...prev };
      delete next[name];
      return next;
    });
  };

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!selectedDate) newErrors.visit_date = '来店日を選択してください';
    if (!form.visit_time) newErrors.visit_time = '来店時間を選択してください';
    if (!form.name.trim()) newErrors.name = 'お名前を入力してください';
    if (form.name.length > 100) newErrors.name = 'お名前は100文字以内で入力してください';
    if (!form.phone.trim()) newErrors.phone = '電話番号を入力してください';
    if (!/^[\d\-+() ]+$/.test(form.phone)) newErrors.phone = '電話番号の形式が正しくありません';
    if (!form.email.trim()) newErrors.email = 'メールアドレスを入力してください';
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) newErrors.email = 'メールアドレスの形式が正しくありません';
    if (form.note.length > 500) newErrors.note = '備考は500文字以内で入力してください';

    const people = parseInt(form.people);
    if (people < 1 || people > 7) newErrors.people = '人数は1〜7名で指定してください';

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!validate()) return;

    setIsSubmitting(true);

    const payload: ReservationRequest = {
      name: form.name.trim(),
      people: parseInt(form.people),
      visit_date: selectedDate!,
      visit_time: form.visit_time,
      phone: form.phone.trim(),
      email: form.email.trim(),
      note: form.note.trim(),
      recaptcha_token: 'mock-token', // TODO: reCAPTCHAトークンを取得
    };

    try {
      // TODO: POST /api/v1/reservations に置き換え
      console.log('予約申請:', payload);

      // モック: 1秒待ってから完了画面に遷移
      await new Promise((resolve) => setTimeout(resolve, 1000));

      // 完了画面に予約情報を渡す
      sessionStorage.setItem('reservation_complete', JSON.stringify({
        visit_date: formatDateJa(selectedDate!),
        visit_time: form.visit_time,
        people: form.people,
        name: form.name.trim(),
      }));

      window.location.href = '/reservation/complete';
    } catch {
      setErrors({ submit: '予約の申請に失敗しました。時間をおいて再度お試しください。' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const inputClass = (fieldName: string) =>
    `w-full px-4 py-3 rounded-lg border ${
      errors[fieldName] ? 'border-red-400 bg-red-50/50' : 'border-gray-200 bg-white'
    } focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-colors text-base`;

  return (
    <form onSubmit={handleSubmit} className="space-y-8">
      {/* 来店日 */}
      <section>
        <h2 className="text-lg font-medium mb-3">来店日を選択</h2>
        {availabilityError && (
          <div className="rounded-lg bg-red-50 border border-red-200 p-3 text-red-700 text-sm mb-3">
            {availabilityError}
          </div>
        )}
        {isAvailabilityLoading ? (
          <div className="rounded-xl border border-gray-200 bg-white p-8 text-center text-gray-400">
            空き状況を読み込み中...
          </div>
        ) : availabilityError ? (
          <div className="rounded-xl border border-gray-200 bg-gray-50 p-8 text-center text-gray-500 text-sm">
            空き状況を取得できないため、来店日を選択できません
          </div>
        ) : (
          <Calendar
            selectedDate={selectedDate}
            onSelect={handleDateSelect}
            availabilityMap={availabilityMap}
          />
        )}
        {errors.visit_date && <p className="text-red-500 text-sm mt-2">{errors.visit_date}</p>}
        {selectedDate && (
          <p className="text-sm text-primary mt-2 font-medium">
            選択中: {formatDateJa(selectedDate)}
            {availabilityMap.get(selectedDate) && (
              <span className="ml-2">（残り {availabilityMap.get(selectedDate)!.available} 食）</span>
            )}
          </p>
        )}
      </section>

      {/* 来店時間 */}
      <section>
        <h2 className="text-lg font-medium mb-3">来店時間</h2>
        <select
          name="visit_time"
          value={form.visit_time}
          onChange={handleChange}
          disabled={!selectedDate}
          className={inputClass('visit_time')}
        >
          <option value="">選択してください</option>
          {timeSlots.map((slot) => (
            <option key={slot} value={slot}>
              {slot}
            </option>
          ))}
        </select>
        {errors.visit_time && <p className="text-red-500 text-sm mt-1">{errors.visit_time}</p>}
      </section>

      {/* 人数 */}
      <section>
        <h2 className="text-lg font-medium mb-3">人数</h2>
        <select name="people" value={form.people} onChange={handleChange} className={inputClass('people')}>
          {[1, 2, 3, 4, 5, 6, 7].map((n) => (
            <option key={n} value={n}>
              {n}名
            </option>
          ))}
        </select>
        <p className="text-xs text-gray-400 mt-1">
          ※ 8名以上の場合は
          <a
            href="https://www.instagram.com/satehits/"
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary underline"
          >
            Instagram DM
          </a>
          よりお問い合わせください
        </p>
        {errors.people && <p className="text-red-500 text-sm mt-1">{errors.people}</p>}
      </section>

      {/* お名前 */}
      <section>
        <h2 className="text-lg font-medium mb-3">
          お名前 <span className="text-red-500 text-sm">*</span>
        </h2>
        <input
          type="text"
          name="name"
          value={form.name}
          onChange={handleChange}
          placeholder="山田太郎"
          className={inputClass('name')}
        />
        {errors.name && <p className="text-red-500 text-sm mt-1">{errors.name}</p>}
      </section>

      {/* 電話番号 */}
      <section>
        <h2 className="text-lg font-medium mb-3">
          電話番号 <span className="text-red-500 text-sm">*</span>
        </h2>
        <input
          type="tel"
          name="phone"
          value={form.phone}
          onChange={handleChange}
          placeholder="090-1234-5678"
          className={inputClass('phone')}
        />
        {errors.phone && <p className="text-red-500 text-sm mt-1">{errors.phone}</p>}
      </section>

      {/* メールアドレス */}
      <section>
        <h2 className="text-lg font-medium mb-3">
          メールアドレス <span className="text-red-500 text-sm">*</span>
        </h2>
        <input
          type="email"
          name="email"
          value={form.email}
          onChange={handleChange}
          placeholder="yamada@example.com"
          className={inputClass('email')}
        />
        {errors.email && <p className="text-red-500 text-sm mt-1">{errors.email}</p>}
      </section>

      {/* 備考 */}
      <section>
        <h2 className="text-lg font-medium mb-3">備考</h2>
        <textarea
          name="note"
          value={form.note}
          onChange={handleChange}
          placeholder="魚の火入れ希望・テーブル席希望等"
          rows={3}
          className={inputClass('note')}
        />
        <p className="text-xs text-gray-400 mt-1">
          ※ アレルギー・苦手食材の対応は行っておりません
        </p>
        {errors.note && <p className="text-red-500 text-sm mt-1">{errors.note}</p>}
      </section>

      {/* reCAPTCHA placeholder */}
      <div className="rounded-lg border border-gray-200 bg-gray-50 p-4 flex items-center gap-3">
        <div className="w-7 h-7 border-2 border-gray-300 rounded flex items-center justify-center">
          <svg className="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
        </div>
        <span className="text-sm text-gray-600">reCAPTCHA（モック）</span>
        {/* TODO: Google reCAPTCHA v2/v3 を導入 */}
      </div>

      {/* エラーメッセージ */}
      {errors.submit && (
        <div className="rounded-lg bg-red-50 border border-red-200 p-4 text-red-700 text-sm">
          {errors.submit}
        </div>
      )}

      {/* 送信ボタン */}
      <button
        type="submit"
        disabled={isSubmitting || isAvailabilityLoading || !!availabilityError}
        className={`w-full py-4 rounded-xl text-white font-medium text-lg transition-all ${
          isSubmitting
            ? 'bg-gray-400 cursor-not-allowed'
            : 'bg-primary hover:bg-primary-dark active:scale-[0.98] shadow-lg hover:shadow-xl'
        }`}
      >
        {isSubmitting ? (
          <span className="flex items-center justify-center gap-2">
            <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            送信中...
          </span>
        ) : (
          '予約を申請する'
        )}
      </button>
    </form>
  );
}
