import { EventEmitter } from 'events';

import { useCallback, useEffect } from 'react';

export enum KeyEvent {
  KeyUp = 'KeyUp',
  KeyDown = 'KeyDown'
}

export enum KeyCode {
  Space = 'Space',
  Escape = 'Escape',
}

export const keyEmitter = new EventEmitter();

const specialKeyCodes = new Set<KeyCode>([
  KeyCode.Escape,
]);

let listenerCount = 0;

const useKeyTracker = (): void => {
  const handleKeyUp = useCallback((e: KeyboardEvent) => {
    if (e.target && !specialKeyCodes.has(e.code as KeyCode)) {
      const element = e.target as Element;
      if ([ 'input', 'textarea' ].includes(element.tagName.toLowerCase())) return;
      if (element.getAttribute('contenteditable')) return;
    }
    keyEmitter.emit(KeyEvent.KeyUp, e);
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.target && !specialKeyCodes.has(e.code as KeyCode)) {
      const element = e.target as Element;
      if ([ 'input', 'textarea' ].includes(element.tagName.toLowerCase())) return;
      if (element.getAttribute('contenteditable')) return;
    }
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
  }, [ handleKeyUp, handleKeyDown ]);
};

export default useKeyTracker;
