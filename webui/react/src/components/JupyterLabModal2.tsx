/* eslint-disable @typescript-eslint/no-unused-vars */
import Button from 'hew/Button';
import CodeEditor from 'hew/CodeEditor';
import Column from 'hew/Column';
import Input from 'hew/Input';
import { Modal } from 'hew/Modal';
import Row from 'hew/Row';
import Select, { OptGroup, Option } from 'hew/Select';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { XOR } from 'hew/utils/types';
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

interface ConfigInputs {
  config: string | Loadable<string>;
  workspaceId: number;
}

interface WizardInputs {
  template?: string;
  name?: string;
  slots?: number;
  pool?: string;
  workspaceId: number;
}

type FormInputs = XOR<WizardInputs, ConfigInputs>;

//const CodeEditor = React.lazy(() => import('hew/CodeEditor'));

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
    register,
    handleSubmit: handleWizardSubmit,
    watch,
    control,
    setValue: setWizardValue,
    formState: { isValid },
  } = useForm<WizardInputs>();

  const values = watch();

  const { settings: defaults, updateSettings: updateDefaults } =
    useSettings<JupyterLabOptions>(settingsConfig);

  const handleModalClose = useCallback(() => {
    updateDefaults(values);
  }, [updateDefaults, values]);

  const fetchConfig = useCallback(async () => {
    setConfig(NotLoaded);
    try {
      const newConfig = await previewJupyterLab({
        name: values.name,
        pool: values.pool,
        slots: values.slots,
        template: values.template,
        workspaceId: values.workspaceId,
      });
      setConfig(Loaded(yaml.dump(newConfig)));
    } catch (e) {
      setConfigError('Unable to fetch JupyterLab config.');
    }
  }, [values.name, values.pool, values.slots, values.template, values.workspaceId]);

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
          pool: fields?.pool,
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

  //~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  const [templates, setTemplates] = useState<Template[]>([]);

  const resourcePools = useObservable(clusterStore.resourcePools).getOrElse([]);

  const resourceInfo = useMemo(() => {
    const selectedPool = resourcePools.find((pool) => pool.name === values.name);
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
  }, [resourcePools, values.name]);

  useEffect(() => {
    if (resourceInfo.hasCompute) {
      if (values.slots === undefined || isNaN(values.slots))
        setWizardValue('slots', DEFAULT_SLOT_COUNT);
    } else if (resourceInfo.hasAux) setWizardValue('slots', 0);
  }, [resourceInfo, setWizardValue, values.slots]);

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
    return configError === undefined && (!!workspace || values.workspaceId);
  }, [configError, values.workspaceId, workspace]);

  //~~~~~~~~~~~~~~~~~~~~~~~

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
      <form id={[idPrefix, BASE_FORM_ID].join('-')} onSubmit={handleWizardSubmit(onSubmit)}>
        <Column>
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
                defaultValue={defaults?.template}
                name="template"
                render={({ field }) => (
                  <Select
                    allowClear
                    label="Template"
                    placeholder="No template (optional)"
                    {...field}>
                    {templates.map((temp) => (
                      <Option key={temp.name} value={temp.name}>
                        {temp.name}
                      </Option>
                    ))}
                  </Select>
                )}
              />
              <Row>
                <label htmlFor="name">Name</label>
                <Controller
                  control={control}
                  name="name"
                  render={({ field }) => (
                    <Input id="name" placeholder="Name (optional)" {...field} />
                  )}
                />
              </Row>
              <Controller
                control={control}
                defaultValue={defaults?.pool ?? resourcePools[0]?.name}
                name="pool"
                render={({ field }) => (
                  <Select allowClear label="Pool" placeholder="Pick the best option" {...field}>
                    <Option value="Default">Pick the best option</Option>
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
              <input
                defaultValue={undefined}
                type={resourceInfo.hasCompute ? 'number' : 'hidden'}
                {...register('slots', {
                  max:
                    resourceInfo.maxSlots === -1 ? Number.MAX_SAFE_INTEGER : resourceInfo.maxSlots,
                  min: 0,
                  valueAsNumber: true,
                })}
              />
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
