import Input from 'hew/Input';
import { Modal, ModalCloseReason } from 'hew/Modal';
import Row from 'hew/Row';
import Select, { SelectValue } from 'hew/Select';
import Tags, { tagsActionHelper } from 'hew/Tags';
import { useToast } from 'hew/Toast';
import { Title } from 'hew/Typography';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback, useMemo, useState } from 'react';

import Link from 'components/Link';
import EditableMetadata from 'components/Metadata/EditableMetadata';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { postModelVersion } from 'services/api';
import { Metadata, ModelItem } from 'types';
import { ensureArray } from 'utils/data';
import handleError, { ErrorType } from 'utils/error';
import { pluralizer } from 'utils/string';

import css from './RegisterCheckpointModal.module.scss';

interface ModalProps {
  checkpoints: string | string[];
  closeModal: (reason: ModalCloseReason) => void;
  models: Loadable<ModelItem[]>;
  modelName?: string;
  openModelModal: () => void;
}

interface ModalState {
  expandDetails: boolean;
  metadata: Metadata;
  selectedModelName?: string;
  tags: string[];
  versionDescription: string;
  versionName: string;
}

const INITIAL_MODAL_STATE = {
  expandDetails: false,
  metadata: {},
  tags: [],
  versionDescription: '',
  versionName: '',
};

const RegisterCheckpointModal: React.FC<ModalProps> = ({
  checkpoints,
  closeModal,
  openModelModal,
  models,
  modelName,
}) => {
  const { openToast } = useToast();
  const { canCreateModelVersion } = usePermissions();
  const [modalState, setModalState] = useState<ModalState>({
    ...INITIAL_MODAL_STATE,
    selectedModelName: modelName,
  });
  const { selectedModelName } = modalState;
  const checkpointsArr = useMemo(() => ensureArray(checkpoints), [checkpoints]);

  const selectedModelNumVersions = useMemo(() => {
    return (
      Loadable.getOrElse([], models).find((model) => model.name === selectedModelName)
        ?.numVersions ?? 0
    );
  }, [models, selectedModelName]);

  const modelOptions = useMemo(() => {
    return Loadable.getOrElse([], models)
      .filter((model) => canCreateModelVersion({ model }))
      .map((model) => ({ id: model.id, name: model.name }));
  }, [models, canCreateModelVersion]);

  const registerModelVersion = useCallback(
    async (state: ModalState) => {
      const { versionDescription, tags, metadata, versionName } = state;
      if (!selectedModelName || !checkpointsArr) return;
      try {
        if (checkpointsArr.length === 1) {
          const response = await postModelVersion({
            body: {
              checkpointUuid: checkpointsArr[0],
              comment: versionDescription,
              labels: tags,
              metadata,
              modelName: selectedModelName,
              name: versionName,
            },
            modelName: selectedModelName,
          });

          if (!response) return;

          openToast({
            description: `"${versionName || `Version ${selectedModelNumVersions + 1}`} registered"`,
            link: (
              <Link path={paths.modelVersionDetails(selectedModelName, response.version)}>
                View Model Version
              </Link>
            ),
            title: 'Version Registered',
          });
        } else {
          for (const checkpointUuid of checkpointsArr) {
            await postModelVersion({
              body: {
                checkpointUuid,
                comment: versionDescription,
                labels: tags,
                metadata,
                modelName: selectedModelName,
              },
              modelName: selectedModelName,
            });
          }
          openToast({
            description: `${checkpointsArr.length} versions registered`,
            link: <Link path={paths.modelDetails(selectedModelName)}>View Model</Link>,
            title: 'Versions Registered',
          });
        }
      } catch (e) {
        handleError(e, {
          publicSubject: `Unable to register ${pluralizer(checkpointsArr.length, 'checkpoint')}.`,
          silent: false,
          type: ErrorType.Api,
        });
      }
    },
    [checkpointsArr, selectedModelNumVersions, selectedModelName, openToast],
  );

  const handleOk = useCallback(async () => {
    await registerModelVersion(modalState);
  }, [registerModelVersion, modalState]);

  const updateModel = useCallback((value: SelectValue) => {
    setModalState((prev) => ({ ...prev, selectedModelName: value?.toString() }));
  }, []);

  const updateVersionName = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setModalState((prev) => ({ ...prev, versionName: e.target.value }));
  }, []);

  const updateVersionDescription = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setModalState((prev) => ({ ...prev, versionDescription: e.target.value }));
  }, []);

  const openDetails = useCallback(() => {
    setModalState((prev) => ({ ...prev, expandDetails: true }));
  }, []);

  const updateMetadata = useCallback((value: Metadata) => {
    setModalState((prev) => ({ ...prev, metadata: value }));
  }, []);

  const updateTags = useCallback((value: string[]) => {
    setModalState((prev) => ({ ...prev, tags: value }));
  }, []);

  const launchNewModelModal = useCallback(() => {
    openModelModal();
    closeModal('ok');
  }, [openModelModal, closeModal]);

  const { expandDetails, metadata, tags, versionDescription, versionName } = modalState;

  return (
    <Modal
      submit={{
        disabled: selectedModelName === undefined,
        handleError,
        handler: handleOk,
        text: 'Register Checkpoint',
      }}
      title="Register Checkpoint">
      <div className={css.base}>
        <p className={css.directions}>Save this checkpoint to the Model Registry</p>
        <div>
          <Row justifyContent="space-between">
            <Title size="x-small">Select Model</Title>
            <Link onClick={() => launchNewModelModal()}>New Model</Link>
          </Row>
          <Select
            options={modelOptions.map((option) => ({ label: option.name, value: option.name }))}
            placeholder="Select a model..."
            value={selectedModelName}
            width={'100%'}
            onChange={updateModel}
          />
        </div>
        {selectedModelName && (
          <>
            <div className={css.separator} />
            <div>
              <h2>Version Name</h2>
              <Input
                disabled={checkpointsArr.length > 1}
                placeholder={`Version ${selectedModelNumVersions + 1}`}
                value={versionName}
                onChange={updateVersionName}
              />
              {checkpointsArr.length > 1 && (
                <p>Cannot specify version name when batch registering.</p>
              )}
            </div>
            <div>
              <h2>
                Description <span>(optional)</span>
              </h2>
              <Input.TextArea value={versionDescription} onChange={updateVersionDescription} />
            </div>
            {expandDetails ? (
              <>
                <div>
                  <h2>
                    Metadata <span>(optional)</span>
                  </h2>
                  <EditableMetadata
                    editing={true}
                    metadata={metadata}
                    updateMetadata={updateMetadata}
                  />
                </div>
                <div>
                  <h2>
                    Tags <span>(optional)</span>
                  </h2>
                  <Tags tags={tags} onAction={tagsActionHelper(tags, updateTags)} />
                </div>
              </>
            ) : (
              <p className={css.expandDetails} onClick={openDetails}>
                Add More Details...
              </p>
            )}
          </>
        )}
      </div>
    </Modal>
  );
};

export default RegisterCheckpointModal;
