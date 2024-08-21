import Icon from 'hew/Icon';
import Spinner from 'hew/Spinner';
import { Failed, Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { TreeNode } from 'hew/utils/types';
import yaml from 'js-yaml';
import React, { useMemo } from 'react';

import { useAsync } from 'hooks/useAsync';
import { paths } from 'routes/utils';
import { getExperimentFileFromTree, getExperimentFileTree } from 'services/api';
import { V1FileNode } from 'services/api-ts-sdk';
import { ExperimentBase, RawJson } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { isSingleTrialExperiment } from 'utils/experiment';

import css from './ExperimentCodeViewer.module.scss';

const CodeEditor = React.lazy(() => import('hew/CodeEditor'));

const configIcon = <Icon name="settings" title="settings" />;

export interface Props {
  experiment: ExperimentBase;
  onSelectFile: (arg0: string) => void;
  selectedFilePath: string;
}

const ExperimentCodeViewer: React.FC<Props> = ({
  experiment,
  onSelectFile,
  selectedFilePath,
}: Props) => {
  const submittedConfig = useMemo(() => {
    if (!experiment.originalConfig) return;

    const { hyperparameters, ...restConfig } = yaml.load(experiment.originalConfig) as RawJson;

    // don't ask me why this works.. it gets rid of the JSON though
    return yaml.dump({ ...restConfig, hyperparameters });
  }, [experiment.originalConfig]);

  const runtimeConfig = useMemo(() => {
    if (!experiment.configRaw) return;

    const {
      environment: { registry_auth, ...restEnvironment },
      workspace,
      project,
      ...restConfig
    } = experiment.configRaw;
    return yaml.dump({ environment: restEnvironment, ...restConfig });
  }, [experiment.configRaw]);

  const expFiles = useAsync<TreeNode[]>(async () => {
    const convertV1FileNodeToTreeNode = (node: V1FileNode): TreeNode => ({
      children: node.files?.map((n) => convertV1FileNodeToTreeNode(n)) ?? [],
      download: paths.experimentFileFromTree(experiment.id, String(node.path)),
      isLeaf: !node.isDir,
      key: node.path ?? '',
      title: node.name,
    });

    const fileTree = await getExperimentFileTree({ experimentId: experiment.id });
    return fileTree.map<TreeNode>(convertV1FileNodeToTreeNode);
  }, [experiment.id]);

  // handle blank selected file path
  const filePath =
    selectedFilePath ||
    (submittedConfig && 'Submitted Configuration') ||
    (runtimeConfig && 'Runtime Configuration') ||
    '';

  const keyedFileContent = useAsync<[string, string]>(async () => {
    if (filePath === 'Submitted Configuration' && submittedConfig !== undefined) {
      return [filePath, submittedConfig];
    }
    if (filePath === 'Runtime Configuration' && runtimeConfig !== undefined) {
      return [filePath, runtimeConfig];
    }
    try {
      const file = await getExperimentFileFromTree({
        experimentId: experiment.id,
        path: filePath,
      });
      if (!file) {
        return Failed(new Error('File has no content'));
      } else {
        return [filePath, file];
      }
    } catch (error) {
      handleError(error, {
        publicMessage: 'Failed to load selected file.',
        publicSubject: 'Unable to fetch the selected file.',
        silent: false,
        type: ErrorType.Api,
      });
      return Failed(new Error('Unable to fetch file.'));
    }
  }, [experiment.id, filePath, runtimeConfig, submittedConfig]);

  const fileContent = useMemo(() => {
    return keyedFileContent.flatMap(([key, content]) => {
      if (key !== filePath) {
        return NotLoaded;
      }
      return Loaded(content);
    });
  }, [filePath, keyedFileContent]);

  const fileOpts = [
    submittedConfig
      ? {
          download: `${experiment.id}_submitted_configuration.yaml`,
          icon: configIcon,
          isLeaf: true,
          key: 'Submitted Configuration',
          subtitle: 'original submitted config',
          title: 'Submitted Configuration',
        }
      : null,
    runtimeConfig
      ? {
          download: `${experiment.id}_runtime_configuration.yaml`,
          icon: configIcon,
          isLeaf: true,
          key: 'Runtime Configuration',
          subtitle: 'after merge with defaults and templates',
          title: 'Runtime Configuration',
        }
      : null,
    ...Loadable.getOrElse([], expFiles),
  ].filter((valid) => !!valid) as TreeNode[];

  const cssClasses = [
    css.codeContainer,
    isSingleTrialExperiment(experiment) || css.multitrialContainer,
  ];

  return (
    <React.Suspense fallback={<Spinner spinning tip="Loading code viewer..." />}>
      <Spinner spinning={expFiles === NotLoaded} tip="Loading file tree...">
        <div className={cssClasses.join(' ')}>
          <CodeEditor
            file={fileContent}
            files={fileOpts}
            readonly={true}
            selectedFilePath={filePath}
            onError={handleError}
            onSelectFile={onSelectFile}
          />
        </div>
      </Spinner>
    </React.Suspense>
  );
};

export default ExperimentCodeViewer;
