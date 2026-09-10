import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// globals: true でも明示的に cleanup して、テスト間で DOM が残らないようにする
afterEach(() => {
  cleanup();
});
