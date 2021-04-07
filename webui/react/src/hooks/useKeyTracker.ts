import { EventEmitter } from 'events';

import { useCallback, useEffect, useRef } from 'react';

export enum KeyEvent {
  KeyUp = 'KeyUp'
}

export const keyEmitter = new EventEmitter();

const useKeyTracker = (): void => {
  const emitter = useRef(keyEmitter);

  const handleKeyUp = useCallback((e: KeyboardEvent) => {
    if (e.target) {
      const element = e.target as Element;
      if ([ 'input', 'textarea' ].includes(element.tagName.toLowerCase())) return;
      if (element.getAttribute('contenteditable')) return;
    }
    emitter.current.emit(KeyEvent.KeyUp, e);
  }, []);

  useEffect(() => {
    document.body.addEventListener('keyup', handleKeyUp);

    return () => {
      document.body.removeEventListener('keyup', handleKeyUp);
    };
  }, [ handleKeyUp ]);
};

export default useKeyTracker;
