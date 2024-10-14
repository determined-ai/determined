import Alert from 'hew/Alert';
import Button from 'hew/Button';
import CodeEditor from 'hew/CodeEditor';
import Column from 'hew/Column';
import Form, { hasErrors } from 'hew/Form';
import Pivot, { PivotProps } from 'hew/Pivot';
import Row from 'hew/Row';
import Spinner from 'hew/Spinner';
import { useToast } from 'hew/Toast';
import useConfirm from 'hew/useConfirm';
import { Loadable, NotLoaded } from 'hew/utils/loadable';
import yaml from 'js-yaml';
import { isEmpty } from 'lodash';
import { useState } from 'react';

import Link from 'components/Link';
import { useAsync } from 'hooks/useAsync';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import {
  deleteGlobalConfigPolicies,
  deleteWorkspaceConfigPolicies,
  getGlobalConfigPolicies,
  getWorkspaceConfigPolicies,
  updateGlobalConfigPolicies,
  updateWorkspaceConfigPolicies,
} from 'services/api';
import { XOR } from 'types';
import handleError from 'utils/error';

type Props = XOR<{ workspaceId: number }, { global: true }>;

type ConfigPoliciesType = 'experiments' | 'tasks';

interface ConfigPoliciesValues {
  label: string;
  workloadType: 'NTSC' | 'EXPERIMENT';
}

const ConfigPoliciesValues: Record<ConfigPoliciesType, ConfigPoliciesValues> = {
  experiments: {
    label: 'Experiments',
    workloadType: 'EXPERIMENT',
  },
  tasks: {
    label: 'Tasks',
    workloadType: 'NTSC',
  },
};

type TabProps = Props & {
  type: ConfigPoliciesType;
};

type FormInputs = {
  [YAML_FORM_ITEM_NAME]: string;
};

const SUCCESS_MESSAGE = 'Configuration policies updated.';
const YAML_FORM_ITEM_NAME = 'configPolicies';

const ConfigPolicies: React.FC<Props> = (props: Props) => {
  const tabItems: PivotProps['items'] = [
    {
      children: <ConfigPoliciesTab type="experiments" {...props} />,
      key: 'experiments',
      label: ConfigPoliciesValues.experiments.label,
    },
    {
      children: <ConfigPoliciesTab type="tasks" {...props} />,
      key: 'tasks',
      label: ConfigPoliciesValues.tasks.label,
    },
  ];

  return <Pivot items={tabItems} type="secondary" />;
};

const ConfigPoliciesTab: React.FC<TabProps> = ({ workspaceId, global, type }: TabProps) => {
  const confirm = useConfirm();
  const { openToast } = useToast();
  const {
    canModifyWorkspaceConfigPolicies,
    canModifyGlobalConfigPolicies,
    loading: rbacLoading,
  } = usePermissions();
  const [form] = Form.useForm<FormInputs>();

  const [disabled, setDisabled] = useState(true);

  const applyMessage = global
    ? "You're about to apply these configuration policies to the cluster."
    : "You're about to apply these configuration policies to the workspace.";
  const viewMessage = global
    ? 'Global configuration policies are being applied to the cluster.'
    : 'Global configuration policies are being applied to the workspace.';
  const confirmMessageEnding = global
    ? 'underlying workspaces, projects, and submitted experiments in the cluster.'
    : 'underlying projects and their experiments in this workspace.';

  const updatePolicies = async () => {
    const configPolicies = form.getFieldValue(YAML_FORM_ITEM_NAME);
    const workloadType = ConfigPoliciesValues[type].workloadType;

    try {
      if (global) {
        configPolicies.length
          ? await updateGlobalConfigPolicies({ configPolicies, workloadType })
          : await deleteGlobalConfigPolicies({ workloadType });
      } else if (workspaceId) {
        configPolicies.length
          ? await updateWorkspaceConfigPolicies({ configPolicies, workloadType, workspaceId })
          : await deleteWorkspaceConfigPolicies({ workloadType, workspaceId });
      }
      openToast({ title: SUCCESS_MESSAGE });
    } catch (error) {
      handleError(error);
    }
  };

  const confirmApply = () => {
    confirm({
      content: (
        <span>
          This will impact{' '}
          <strong>
            <u>all</u>
          </strong>{' '}
          {confirmMessageEnding}
        </span>
      ),
      okText: 'Apply',
      onConfirm: updatePolicies,
      onError: handleError,
      size: 'medium',
      title: applyMessage,
    });
  };

  const loadableConfigPolicies: Loadable<string | undefined> = useAsync(async () => {
    if (global) {
      const response = await getGlobalConfigPolicies({
        workloadType: ConfigPoliciesValues[type].workloadType,
      });
      if (isEmpty(response.configPolicies)) return undefined;
      return response.configPolicies;
    } else if (workspaceId) {
      const response = await getWorkspaceConfigPolicies({
        workloadType: ConfigPoliciesValues[type].workloadType,
        workspaceId,
      });
      if (isEmpty(response.configPolicies)) return undefined;
      return response.configPolicies;
    }
    return NotLoaded;
  }, [workspaceId, type, global]);

  const initialYAML = yaml.dump(loadableConfigPolicies.getOrElse(undefined));

  const canModify = global ? canModifyGlobalConfigPolicies : canModifyWorkspaceConfigPolicies;

  const handleChange = () => {
    setDisabled(hasErrors(form) || form.getFieldValue(YAML_FORM_ITEM_NAME) === initialYAML);
  };

  const docsLink = (
    <Link external path={paths.docs('/manage/config-policies.html')} popout>
      Learn more
    </Link>
  );

  if (rbacLoading) return <Spinner spinning />;

  return (
    <Column>
      <Row width="fill">
        <div style={{ width: '100%' }}>
          {canModify ? (
            <Alert
              action={
                <Button disabled={disabled} onClick={confirmApply}>
                  Apply
                </Button>
              }
              description={docsLink}
              message={applyMessage}
              showIcon
            />
          ) : (
            <Alert
              description={docsLink}
              message={viewMessage}
              showIcon
            />
          )}
        </div>
      </Row>
      <Row width="fill">
        <div style={{ width: '100%' }}>
          <Form form={form} onFieldsChange={handleChange}>
            <Form.Item
              name={YAML_FORM_ITEM_NAME}
              rules={[
                {
                  validator: (_, value) => {
                    try {
                      yaml.load(value);
                      return Promise.resolve();
                    } catch (err: unknown) {
                      return Promise.reject(
                        new Error(
                          `Invalid YAML on line ${(err as { mark: { line: string } }).mark.line}.`,
                        ),
                      );
                    }
                  },
                },
              ]}>
              <CodeEditor
                file={initialYAML}
                files={[{ key: type, title: `${type}-config-policies.yaml` }]}
                readonly={!canModify}
                onError={(error) => {
                  handleError(error);
                }}
              />
            </Form.Item>
          </Form>
        </div>
      </Row>
    </Column>
  );
};

export default ConfigPolicies;
