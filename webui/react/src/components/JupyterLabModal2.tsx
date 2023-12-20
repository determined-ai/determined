import Button from 'hew/Button';
import CodeEditor from 'hew/CodeEditor';
import Column from 'hew/Column';
import Input from 'hew/Input';
import InputNumber from 'hew/InputNumber';
import { Modal } from 'hew/Modal';
import Select, { OptGroup, Option } from 'hew/Select';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { number, string, undefined as undefinedType, union } from 'io-ts';
import yaml from 'js-yaml';
import React, { useCallback, useEffect, useId, useMemo, useState } from 'react';
import { Controller, SubmitHandler, useForm } from 'react-hook-form';

import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getTaskTemplates } from 'services/api';
import clusterStore from 'stores/cluster';
import workspaceStore from 'stores/workspaces';
import { RawJson, Template, Workspace } from 'types';
import handleError from 'utils/error';
import { JupyterLabOptions, launchJupyterLab, previewJupyterLab } from 'utils/jupyter';
import { useObservable } from 'utils/observable';

const STORAGE_PATH = 'jupyter-lab';
const DEFAULT_SLOT_COUNT = 1;
const BASE_FORM_ID = 'jupyter-form';

const settingsConfig: SettingsConfig<JupyterLabOptions> = {
  settings: {
    name: {
      defaultValue: '',
      skipUrlEncoding: true,
      storageKey: 'name',
      type: union([string, undefinedType]),
    },
    pool: {
      defaultValue: '',
      skipUrlEncoding: true,
      storageKey: 'pool',
      type: union([string, undefinedType]),
    },
    slots: {
      defaultValue: DEFAULT_SLOT_COUNT,
      skipUrlEncoding: true,
      storageKey: 'slots',
      type: union([number, undefinedType]),
    },
    template: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'template',
      type: union([string, undefinedType]),
    },
    workspaceId: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'workspaceId',
      type: union([number, undefinedType]),
    },
  },
  storagePath: STORAGE_PATH,
};

interface Props {
  workspace?: Workspace;
}

interface FormInputs {
  template?: string;
  name?: string;
  slots?: number;
  pool?: string;
  workspaceId: number;
}

const JupyterLabModalComponent: React.FC<Props> = ({ workspace }: Props) => {
  const idPrefix = useId();
  const [showFullConfig, setShowFullConfig] = useState(false);
  const [config, setConfig] = useState<Loadable<string>>(NotLoaded);
  const [configError, setConfigError] = useState<string>();
  const { canCreateWorkspaceNSC } = usePermissions();
  const workspaces = useObservable(workspaceStore.workspaces)
    .getOrElse([])
    .filter((workspace) => !workspace.archived && canCreateWorkspaceNSC({ workspace }));

  const {
    handleSubmit,
    watch,
    control,
    setValue,
    formState: { isValid },
  } = useForm<FormInputs>();

  const formValues = watch();

  const { settings: defaults, updateSettings: updateDefaults } =
    useSettings<JupyterLabOptions>(settingsConfig);

  const handleModalClose = useCallback(() => {
    updateDefaults(formValues);
  }, [updateDefaults, formValues]);

  const fetchConfig = useCallback(async () => {
    setConfig(NotLoaded);
    try {
      const newConfig = await previewJupyterLab({
        name: formValues.name,
        pool:
          formValues.pool === [BASE_FORM_ID, 'defaultPool'].join('-') ? undefined : formValues.pool,
        slots: formValues.slots,
        template: formValues.template,
        workspaceId: formValues.workspaceId,
      });
      setConfig(Loaded(yaml.dump(newConfig)));
    } catch (e) {
      setConfigError('Unable to fetch JupyterLab config.');
    }
  }, [
    formValues.name,
    formValues.pool,
    formValues.slots,
    formValues.template,
    formValues.workspaceId,
  ]);

  const handleSecondary = useCallback(() => {
    setShowFullConfig((show) => !show);
  }, []);

  const onSubmit: SubmitHandler<FormInputs> = useCallback(
    async (fields) => {
      updateDefaults(fields);
      if (showFullConfig) {
        await launchJupyterLab({
          config: yaml.load(config.getOrElse('')) as RawJson,
          workspaceId: fields.workspaceId,
        });
      } else {
        await launchJupyterLab({
          name: fields?.name,
          pool: fields?.pool === [BASE_FORM_ID, 'defaultPool'].join('-') ? undefined : fields?.pool,
          slots: fields?.slots,
          template: fields?.template,
          workspaceId: fields.workspaceId,
        });
      }
    },
    [config, showFullConfig, updateDefaults],
  );

  useEffect(() => workspaceStore.fetch(), []);

  // Fetch full config when showing advanced mode.
  useEffect(() => {
    if (showFullConfig) {
      fetchConfig();
    }
  }, [fetchConfig, showFullConfig]);

  const [templates, setTemplates] = useState<Template[]>([]);

  const resourcePools = useObservable(clusterStore.resourcePools).getOrElse([]);

  const resourceInfo = useMemo(() => {
    const selectedPool = resourcePools.find((pool) => pool.name === formValues.pool);
    if (!selectedPool) return { hasAux: false, hasCompute: false, maxSlots: 0 };

    /**
     * For static resource pools, the slots-per-agent comes through as -1,
     * meaning it is unknown how many we may have.
     */
    const hasAuxCapacity = selectedPool.auxContainerCapacityPerAgent > 0;
    const hasSlots = selectedPool.slotsAvailable > 0;
    const maxSlots = selectedPool.slotsPerAgent ?? 0;
    const hasSlotsPerAgent = maxSlots !== 0;
    const hasComputeCapacity = hasSlots || hasSlotsPerAgent;

    return {
      hasAux: hasAuxCapacity,
      hasCompute: hasComputeCapacity,
      maxSlots: maxSlots,
    };
  }, [resourcePools, formValues.pool]);

  useEffect(() => {
    if (resourceInfo.hasCompute) {
      if (formValues.slots === undefined || isNaN(formValues.slots))
        setValue('slots', DEFAULT_SLOT_COUNT);
    } else if (resourceInfo.hasAux) setValue('slots', 0);
  }, [resourceInfo, setValue, formValues.slots]);

  const fetchTemplates = useCallback(async () => {
    try {
      setTemplates(await getTaskTemplates({}));
    } catch (e) {
      handleError(e);
    }
  }, []);

  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  const validateConfig = useCallback((config: string) => {
    try {
      yaml.load(config);
      setConfigError(undefined);
    } catch (err: unknown) {
      setConfigError(`Invalid YAML on line ${(err as { mark: { line: string } }).mark.line}.`);
    }
  }, []);

  const isConfigValid = useMemo(() => {
    return configError === undefined && (!!workspace || formValues.workspaceId);
  }, [configError, formValues.workspaceId, workspace]);

  return (
    <Modal
      cancel
      footerLink={
        showFullConfig ? (
          <Link
            external
            path={paths.docs('/architecture/introduction.html#interactive-job-configuration')}
            popout>
            Read about JupyterLab settings
          </Link>
        ) : undefined
      }
      size={showFullConfig ? 'large' : 'small'}
      submit={{
        disabled: showFullConfig ? !isConfigValid : !isValid,
        form: [idPrefix, BASE_FORM_ID].join('-'),
        handleError,
        handler: () => undefined,
        text: 'Launch',
      }}
      title="Launch JupyterLab"
      onClose={handleModalClose}>
      <form id={[idPrefix, BASE_FORM_ID].join('-')} onSubmit={handleSubmit(onSubmit)}>
        <Column align="stretch">
          <Controller
            control={control}
            defaultValue={workspace?.id}
            disabled={!!workspace}
            name="workspaceId"
            render={({ field }) => (
              <Select
                allowClear
                label="Workspace"
                placeholder="Workspace (required)"
                width="100%"
                {...field}>
                {workspaces.map((workspace: Workspace) => (
                  <Option key={workspace.id} value={workspace.id}>
                    {workspace.name}
                  </Option>
                ))}
              </Select>
            )}
            rules={{ required: true }}
          />
          {showFullConfig ? (
            <>
              <CodeEditor
                file={config}
                files={[{ key: 'config.yaml' }]}
                height="40vh"
                onChange={validateConfig}
                onError={handleError}
              />
              {configError && <p>{configError}</p>}
            </>
          ) : (
            <>
              <Controller
                control={control}
                defaultValue={defaults.template}
                name="template"
                render={({ field }) => (
                  <Select
                    allowClear
                    label="Template"
                    placeholder="No template (optional)"
                    width="100%"
                    {...field}>
                    {templates.map((temp) => (
                      <Option key={temp.name} value={temp.name}>
                        {temp.name}
                      </Option>
                    ))}
                  </Select>
                )}
              />
              <Controller
                control={control}
                name="name"
                render={({ field }) => (
                  <Input id="name" label="Name" placeholder="Name (optional)" {...field} />
                )}
              />
              <Controller
                control={control}
                defaultValue={defaults.pool ?? [BASE_FORM_ID, 'defaultPool'].join('-')}
                name="pool"
                render={({ field }) => (
                  <Select label="Pool" placeholder="Pick the best option" width="100%" {...field}>
                    <Option value={[BASE_FORM_ID, 'defaultPool'].join('-')}>
                      Pick the best option
                    </Option>
                    <OptGroup label="Resource Pools">
                      {resourcePools.map((pool) => (
                        <Option key={pool.name} value={pool.name}>
                          {pool.name}
                        </Option>
                      ))}
                    </OptGroup>
                  </Select>
                )}
              />
              {resourceInfo.hasCompute && (
                <Controller
                  control={control}
                  defaultValue={defaults.slots}
                  name="slots"
                  render={({ field }) => (
                    <InputNumber id="slots" label="Slots" width="100%" {...field} />
                  )}
                  rules={{
                    max:
                      resourceInfo.maxSlots === -1
                        ? Number.MAX_SAFE_INTEGER
                        : resourceInfo.maxSlots,
                    min: 0,
                  }}
                />
              )}
            </>
          )}
        </Column>
      </form>
      <div>
        <Button onClick={handleSecondary}>
          {showFullConfig ? 'Show Simple Config' : 'Show Full Config'}
        </Button>
      </div>
    </Modal>
  );
};

export default JupyterLabModalComponent;
