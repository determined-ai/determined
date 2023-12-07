import Input from 'hew/Input';
import Select, { SelectValue } from 'hew/Select';
import Tags, { tagsActionHelper } from 'hew/Tags';
import { useToast } from 'hew/Toast';
import _ from 'lodash';
import React, { useCallback, useMemo, useState } from 'react';

import Link from 'components/Link';
import EditableMetadata from 'components/Metadata/EditableMetadata';
import { useModal } from 'hew/Modal';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { getModels, postModelVersion } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { Metadata, ModelItem } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { validateDetApiEnum } from 'utils/service';
import { pluralizer } from 'utils/string';

import css from './RegisterCheckpointModal.module.scss';

interface ModalProps {
  modelName?: string;
  onClose: () => void;
}

interface ModalState {
  checkpoints?: string[];
  expandDetails: boolean;
  metadata: Metadata;
  models: ModelItem[];
  selectedModelName?: string;
  tags: string[];
  versionDescription: string;
  versionName: string;
  visible: boolean;
}

const INITIAL_MODAL_STATE = {
  expandDetails: false,
  metadata: {},
  models: [],
  tags: [],
  versionDescription: '',
  versionName: '',
  visible: false,
};

const RegisterCheckpointModal: React.FC<ModalProps> = ({
  onClose,
  modelName: selectedModelName,
}) => {
  const { openToast } = useToast();
  const { canCreateModelVersion } = usePermissions();
  const [models, setModels] = useState<Loadable<ModelItem[]>>(NotLoaded);
  const [modalState, setModalState] = useState<ModalState>(INITIAL_MODAL_STATE);

  const registerModal = useModal(RegisterCheckpointModal);

  const selectedModelNumVersions = useMemo(() => {
    return Loadable.getOrElse([], models).find((model) => model.name === selectedModelName)?.numVersions ??
      0;
  }, [models, selectedModelName]);

  const modelOptions = useMemo(() => {
    return Loadable.getOrElse([], models)
      .filter((model) => canCreateModelVersion({ model }))
      .map((model) => ({ id: model.id, name: model.name }));
  }, [models, canCreateModelVersion]);

  const registerModelVersion = useCallback(
    async (state: ModalState) => {
      const { selectedModelName, versionDescription, tags, metadata, versionName, checkpoints } =
        state;
      if (!selectedModelName || !checkpoints) return;
      try {
        if (checkpoints.length === 1) {
          const response = await postModelVersion({
            body: {
              checkpointUuid: checkpoints[0],
              comment: versionDescription,
              labels: tags,
              metadata,
              modelName: selectedModelName,
              name: versionName,
            },
            modelName: selectedModelName,
          });

          if (!response) return;

          registerModal.close();
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
          for (const checkpointUuid of checkpoints) {
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
          registerModal.close();
          openToast({
            description: `${checkpoints.length} versions registered`,
            link: <Link path={paths.modelDetails(selectedModelName)}>View Model</Link>,
            title: 'Versions Registered',
          });
        }
      } catch (e) {
        handleError(e, {
          publicSubject: `Unable to register ${pluralizer(checkpoints.length, 'checkpoint')}.`,
          silent: true,
          type: ErrorType.Api,
        });
      }
    },
    [selectedModelNumVersions, openToast],
  );

  const handleOk = useCallback(
    async (state: ModalState) => {
      await registerModelVersion(state);
    },
    [registerModelVersion],
  );

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

  const launchNewModelModal = useCallback(
    (state: ModalState) => {
      registerModal.close();
      onClose?.('Cancel', state.checkpoints);
    },
    [onClose, registerModal],
  );

  const fetchModels = useCallback(async () => {
    if (!modalState.visible) return;
    try {
      const response = await getModels(
        {
          archived: false,
          orderBy: 'ORDER_BY_DESC',
          sortBy: validateDetApiEnum(
            V1GetModelsRequestSortBy,
            V1GetModelsRequestSortBy.LASTUPDATEDTIME,
          ),
        },
        { signal: canceler.signal },
      );
      setModels((prev) => {
        const loadedModels = Loaded(response.models);
        if (_.isEqual(prev, loadedModels)) return prev;
        return loadedModels;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch models.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [canceler.signal, modalState.visible]);

  return (
    <div className={css.base}>
      <p className={css.directions}>Save this checkpoint to the Model Registry</p>
      <div>
        <div className={css.selectModelRow}>
          <h2>Select Model</h2>
          <p onClick={() => launchNewModelModal(state)}>New Model</p>
        </div>
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
              disabled={checkpoints?.length != null && checkpoints.length > 1}
              placeholder={`Version ${selectedModelNumVersions + 1}`}
              value={versionName}
              onChange={updateVersionName}
            />
            {checkpoints?.length != null && checkpoints.length > 1 && (
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
  );
};

export default RegisterCheckpointModal;
