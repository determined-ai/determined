import yaml from 'js-yaml';
import React, { useEffect, useMemo, useState } from 'react';

import { ErrorMessage, LoadableOrError } from 'components/kit/CodeEditor';
import Icon from 'components/kit/Icon';
import { TreeNode } from 'components/kit/internal/types';
import Spinner from 'components/kit/Spinner';
import { paths } from 'routes/utils';
import { getExperimentFileFromTree, getExperimentFileTree } from 'services/api';
import { V1FileNode } from 'services/api-ts-sdk';
import { RawJson } from 'types';
import { ExperimentBase } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { isSingleTrialExperiment } from 'utils/experiment';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

import css from './ExperimentCodeViewer.module.scss';

const CodeEditor = React.lazy(() => import('components/kit/CodeEditor'));

const configIcon = <Icon name="settings" title="settings" />;

export interface Props {
  experiment: ExperimentBase;
  onSelectFile?: (arg0: string) => void;
  selectedFilePath?: string;
}

const ExperimentCodeViewer: React.FC<Props> = ({
  experiment,
  onSelectFile,
  selectedFilePath,
}: Props) => {
  const [expFiles, setExpFiles] = useState<Loadable<TreeNode[]>>(NotLoaded);
  const [fileContent, setFileContent] = useState<LoadableOrError<string>>(NotLoaded);

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

  useEffect(() => {
    const convertV1FileNodeToTreeNode = (node: V1FileNode): TreeNode => ({
      children: node.files?.map((n) => convertV1FileNodeToTreeNode(n)) ?? [],
      download: paths.experimentFileFromTree(experiment.id, String(node.path)),
      isLeaf: !node.isDir,
      key: node.path ?? '',
      title: node.name,
    });

    (async () => {
      const fileTree = await getExperimentFileTree({ experimentId: experiment.id });
      setExpFiles(Loaded(fileTree.map<TreeNode>(convertV1FileNodeToTreeNode)));
    })();
  }, [experiment.id]);

  // handle blank selected file path
  const filePath =
    selectedFilePath ||
    (submittedConfig && 'Submitted Configuration') ||
    (runtimeConfig && 'Runtime Configuration');
  useEffect(() => {
    if (filePath === 'Submitted Configuration' && submittedConfig !== undefined) {
      setFileContent(Loaded(submittedConfig));
      return;
    }
    if (filePath === 'Runtime Configuration' && runtimeConfig !== undefined) {
      setFileContent(Loaded(runtimeConfig));
      return;
    }
    setFileContent(NotLoaded);
    (async () => {
      try {
        const file = await getExperimentFileFromTree({
          experimentId: experiment.id,
          path: filePath,
        });
        if (!file) {
          setFileContent(ErrorMessage('File has no content.'));
        } else {
          setFileContent(Loaded(file));
        }
      } catch (error) {
        handleError(error, {
          publicMessage: 'Failed to load selected file.',
          publicSubject: 'Unable to fetch the selected file.',
          silent: false,
          type: ErrorType.Api,
        });
        setFileContent(ErrorMessage('Unable to fetch file.'));
        return;
      }
    })();
  }, [experiment.id, filePath, runtimeConfig, submittedConfig]);

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
