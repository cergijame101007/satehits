import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import TurnstileWidget from '@/components/react/TurnstileWidget';

type RenderOptions = Parameters<NonNullable<Window['turnstile']>['render']>[1];

function stubTurnstile() {
  const turnstile = {
    render: vi.fn<(container: string | HTMLElement, options: RenderOptions) => string>(
      () => 'widget-1',
    ),
    reset: vi.fn(),
    remove: vi.fn(),
  };
  window.turnstile = turnstile;
  return turnstile;
}

function lastRenderOptions(turnstile: ReturnType<typeof stubTurnstile>): RenderOptions {
  const call = turnstile.render.mock.calls.at(-1);
  if (!call) throw new Error('turnstile.render が呼ばれていません');
  return call[1];
}

describe('TurnstileWidget', () => {
  beforeEach(() => {
    delete window.turnstile;
    delete window.onTurnstileLoad;
    document.getElementById('cf-turnstile-script')?.remove();
  });

  afterEach(() => {
    delete window.turnstile;
    delete window.onTurnstileLoad;
    document.getElementById('cf-turnstile-script')?.remove();
  });

  it('siteKey が空なら警告を表示し、ウィジェットを描画しない', () => {
    const turnstile = stubTurnstile();
    render(<TurnstileWidget siteKey="" onToken={() => {}} />);

    expect(screen.getByText('Turnstile のサイトキーが設定されていません')).toBeInTheDocument();
    expect(turnstile.render).not.toHaveBeenCalled();
  });

  it('turnstile が読み込み済みなら siteKey を渡して render する', async () => {
    const turnstile = stubTurnstile();
    const { container } = render(<TurnstileWidget siteKey="site-key" onToken={() => {}} />);

    await waitFor(() => {
      expect(turnstile.render).toHaveBeenCalledTimes(1);
    });
    const [target, options] = turnstile.render.mock.calls[0];
    expect(target).toBe(container.firstElementChild);
    expect(options.sitekey).toBe('site-key');
  });

  it('コールバックで受け取ったトークンを onToken に伝播する', async () => {
    const turnstile = stubTurnstile();
    const onToken = vi.fn();
    render(<TurnstileWidget siteKey="site-key" onToken={onToken} />);

    await waitFor(() => {
      expect(turnstile.render).toHaveBeenCalled();
    });
    lastRenderOptions(turnstile).callback('token-abc');

    expect(onToken).toHaveBeenCalledWith('token-abc');
  });

  it('error-callback でトークンを空にして onError を呼ぶ', async () => {
    const turnstile = stubTurnstile();
    const onToken = vi.fn();
    const onError = vi.fn();
    render(<TurnstileWidget siteKey="site-key" onToken={onToken} onError={onError} />);

    await waitFor(() => {
      expect(turnstile.render).toHaveBeenCalled();
    });
    lastRenderOptions(turnstile)['error-callback']?.();

    expect(onToken).toHaveBeenCalledWith('');
    expect(onError).toHaveBeenCalledTimes(1);
  });

  it('expired-callback でトークンを空にする', async () => {
    const turnstile = stubTurnstile();
    const onToken = vi.fn();
    render(<TurnstileWidget siteKey="site-key" onToken={onToken} />);

    await waitFor(() => {
      expect(turnstile.render).toHaveBeenCalled();
    });
    lastRenderOptions(turnstile)['expired-callback']?.();

    expect(onToken).toHaveBeenCalledWith('');
  });

  it('アンマウント時にウィジェットを remove する', async () => {
    const turnstile = stubTurnstile();
    const { unmount } = render(<TurnstileWidget siteKey="site-key" onToken={() => {}} />);

    await waitFor(() => {
      expect(turnstile.render).toHaveBeenCalled();
    });
    unmount();

    expect(turnstile.remove).toHaveBeenCalledWith('widget-1');
  });

  it('resetKey が変わると再描画する', async () => {
    const turnstile = stubTurnstile();
    const onToken = vi.fn();
    const { rerender } = render(
      <TurnstileWidget siteKey="site-key" onToken={onToken} resetKey={0} />,
    );

    await waitFor(() => {
      expect(turnstile.render).toHaveBeenCalledTimes(1);
    });

    rerender(<TurnstileWidget siteKey="site-key" onToken={onToken} resetKey={1} />);

    await waitFor(() => {
      expect(turnstile.render).toHaveBeenCalledTimes(2);
    });
    expect(turnstile.remove).toHaveBeenCalledWith('widget-1');
  });

  it('turnstile 未読み込みならスクリプトを追加し、読み込み完了後に render する', async () => {
    const onToken = vi.fn();
    render(<TurnstileWidget siteKey="site-key" onToken={onToken} />);

    const script = document.getElementById('cf-turnstile-script') as HTMLScriptElement | null;
    expect(script).not.toBeNull();
    expect(script!.src).toContain('challenges.cloudflare.com/turnstile');

    const turnstile = stubTurnstile();
    script!.onload?.(new Event('load'));

    await waitFor(() => {
      expect(turnstile.render).toHaveBeenCalledTimes(1);
    });
    expect(turnstile.render.mock.calls[0][1].sitekey).toBe('site-key');
  });
});
