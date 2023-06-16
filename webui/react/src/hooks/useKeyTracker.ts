import { EventEmitter } from 'events';

import { useCallback, useEffect } from 'react';

import { ValueOf } from 'types';

export const KeyEvent = {
  KeyDown: 'KeyDown',
  KeyUp: 'KeyUp',
} as const;

export type KeyEvent = ValueOf<typeof KeyEvent>;

export const KeyCode = {
  Escape: 'Escape',
  Space: 'Space',
} as const;

export type KeyCode = ValueOf<typeof KeyCode>;

export const keyEmitter = new EventEmitter();

const specialKeyCodes = new Set<KeyCode>([KeyCode.Escape]);

let listenerCount = 0;

const shouldIgnoreKBEvent = (e: KeyboardEvent): boolean => {
  if (!e.target || specialKeyCodes.has(e.code as KeyCode) || e.ctrlKey || e.altKey || e.metaKey)
    return false;

  const target = e.target as Element;
  if (
    ['input', 'textarea'].includes(target.tagName.toLowerCase()) ||
    !!target.getAttribute('contenteditable')
  ) {
    return true;
  }
  return false;
};

const useKeyTracker = (): void => {
  const handleKeyUp = useCallback((e: KeyboardEvent) => {
    if (shouldIgnoreKBEvent(e)) return;
    keyEmitter.emit(KeyEvent.KeyUp, e);
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (shouldIgnoreKBEvent(e)) return;
    keyEmitter.emit(KeyEvent.KeyDown, e);
  }, []);

  useEffect(() => {
    if (listenerCount !== 0) return;

    listenerCount++;
    document.body.addEventListener('keyup', handleKeyUp);
    document.body.addEventListener('keydown', handleKeyDown);

    return () => {
      if (listenerCount === 0) return;
      document.body.removeEventListener('keyup', handleKeyUp);
      listenerCount--;
    };
  }, [handleKeyUp, handleKeyDown]);
};

export default useKeyTracker;
