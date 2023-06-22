import React, { useEffect } from 'react';

import JupyterLabSettings from 'components/JupyterLab.settings';
import JupyterLabModalComponent from 'components/JupyterLabModal';
import { useModal } from 'components/kit/Modal';
import { keyEmitter, KeyEvent } from 'hooks/useKeyTracker';
import { useSettings } from 'hooks/useSettings';
import { JupyterLabOptions } from 'utils/jupyter';
import { KeyboardShortcut, matchesShortcut } from 'utils/shortcut';

interface Props {
  active?: boolean;
}

const JupyterLabGlobal: React.FC<Props> = ({ active }) => {
  const JupyterLabModal = useModal(JupyterLabModalComponent);
  const { settings } = useSettings<JupyterLabOptions>(JupyterLabSettings);

  useEffect(() => {
    const shortcut: KeyboardShortcut | undefined = settings.shortcut
      ? JSON.parse(settings.shortcut)
      : undefined;

    const keyDownListener = (e: KeyboardEvent) => {
      if (shortcut && matchesShortcut(e, shortcut)) {
        JupyterLabModal.open();
      }
    };

    if (active) keyEmitter.on(KeyEvent.KeyDown, keyDownListener);

    return () => {
      keyEmitter.off(KeyEvent.KeyDown, keyDownListener);
    };
  }, [JupyterLabModal, settings, active]);

  return <JupyterLabModal.Component />;
};

export default JupyterLabGlobal;
