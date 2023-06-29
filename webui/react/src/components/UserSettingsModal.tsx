import { useMemoizedObservable } from 'micro-observables';
import React, { useCallback, useContext, useEffect, useMemo, useState } from 'react';

import { Modal } from 'components/kit/Modal';
import { UserSettings, UserSettingsState } from 'hooks/useSettingsProvider';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

import CodeEditor from './kit/CodeEditor';

interface Props {
  onSave?: () => void;
}

const UserSettingsModal: React.FC<Props> = ({ onSave }: Props) => {
  const { state, isLoading } = useContext(UserSettings);
  const stringifiedState = useMemoizedObservable<string>(
    () => state.select((obj) => JSON.stringify(obj, undefined, ' ')),
    [],
  );
  const [editedSettingsString, setEditedSettingsString] = useState<Loadable<string>>(
    isLoading.get() ? NotLoaded : Loaded(stringifiedState),
  );

  const editedSettings: UserSettingsState | undefined = useMemo(
    () =>
      Loadable.match(editedSettingsString, {
        Loaded: (settingsString) => {
          try {
            return JSON.parse(settingsString);
          } catch {
            return;
          }
        },
        NotLoaded: () => undefined,
      }),
    [editedSettingsString],
  );

  useEffect(() => {
    if (Loadable.isLoaded(editedSettingsString) || isLoading.get()) return;

    setEditedSettingsString(Loaded(stringifiedState));
  }, [isLoading, editedSettingsString, stringifiedState]);

  const handleSave = useCallback(async () => {
    if (!editedSettings) return;

    state.set(editedSettings);
    await onSave?.();
  }, [editedSettings, onSave, state]);

  const handleChange = useCallback((newSettings: string) => {
    setEditedSettingsString(Loaded(newSettings));
  }, []);

  return (
    <Modal
      cancel
      size="medium"
      submit={{
        disabled: editedSettings === undefined,
        handleError,
        handler: handleSave,
        text: 'Save Settings',
      }}
      title="Edit Raw Settings">
      <CodeEditor
        files={[
          {
            content: editedSettingsString,
            key: 'settings.json',
            title: 'settings.json',
          },
        ]}
        height="400px"
        onChange={handleChange}
        onError={handleError}
      />
    </Modal>
  );
};

export default UserSettingsModal;
