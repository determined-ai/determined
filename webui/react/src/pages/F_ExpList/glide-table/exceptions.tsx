import React, { ReactNode } from 'react';

import { ImageAlert } from 'components/Image';
import Button from 'components/kit/Button';
import Link from 'components/Link';
import { paths } from 'routes/utils';
import useUI from 'stores/contexts/UI';
import { DarkLight } from 'utils/themes';

import css from './exceptions.module.scss';

interface ExceptionMessageProps {
  ImageComponent?: React.FC<{ darkLight?: DarkLight }>;
  message?: string;
  children?: ReactNode;
  title: string;
}

const ExperimentImage: React.FC<{ darkLight?: DarkLight }> = () => {
  const {
    ui: { theme: appTheme },
  } = useUI();

  const foreground = appTheme.backgroundOn;
  const background = appTheme.background;
  return (
    <svg fill="none" height="64" viewBox="0 0 60 64" width="60" xmlns="http://www.w3.org/2000/svg">
      <rect height="64" width="60" />
      <g clipPath="url(#clip0_66_8285)">
        <rect height="1024" transform="translate(-690 -407)" width="1440" />
        <mask id="path-1-inside-1_66_8285">
          <path d="M29.9981 28.8878C29.9981 29.7344 30.3344 30.5463 30.933 31.1449C31.5316 31.7435 32.3435 32.0798 33.1901 32.0798C34.0367 32.0798 34.8486 31.7435 35.4472 31.1449C36.0458 30.5463 36.3821 29.7344 36.3821 28.8878C36.3821 28.0412 36.0458 27.2293 35.4472 26.6307C34.8486 26.0321 34.0367 25.6958 33.1901 25.6958C32.3435 25.6958 31.5316 26.0321 30.933 26.6307C30.3344 27.2293 29.9981 28.0412 29.9981 28.8878ZM59.2849 57.0494L44.7053 19.3117V5.42643H50.427V0H9.56916V5.42643H15.2909V19.3117L0.711306 57.0494C0.487865 57.6399 0.368164 58.2623 0.368164 58.8928C0.368164 61.7097 2.65844 64 5.4754 64H54.5208C55.1512 64 55.7736 63.8803 56.3642 63.6569C58.9976 62.6434 60.3063 59.6828 59.2849 57.0494ZM20.7173 20.3252V5.58604H39.2789V20.3252L46.5327 39.1022C44.8809 38.6793 43.1731 38.4638 41.4335 38.4638C36.5497 38.4638 31.9213 40.1796 28.2425 43.2519C25.5273 45.52 22.1005 46.7602 18.5627 46.7551C15.9532 46.7551 13.4475 46.0928 11.237 44.8638L20.7173 20.3252ZM5.93824 58.5736L9.26592 49.9711C12.1148 51.4155 15.2829 52.1895 18.5707 52.1895C23.4544 52.1895 28.0829 50.4738 31.7617 47.4015C34.4669 45.1511 37.8505 43.8983 41.4415 43.8983C44.2345 43.8983 46.8998 44.6564 49.23 46.0609L54.0579 58.5736H5.93824Z" />
        </mask>
        <path
          d="M29.9981 28.8878C29.9981 29.7344 30.3344 30.5463 30.933 31.1449C31.5316 31.7435 32.3435 32.0798 33.1901 32.0798C34.0367 32.0798 34.8486 31.7435 35.4472 31.1449C36.0458 30.5463 36.3821 29.7344 36.3821 28.8878C36.3821 28.0412 36.0458 27.2293 35.4472 26.6307C34.8486 26.0321 34.0367 25.6958 33.1901 25.6958C32.3435 25.6958 31.5316 26.0321 30.933 26.6307C30.3344 27.2293 29.9981 28.0412 29.9981 28.8878ZM59.2849 57.0494L44.7053 19.3117V5.42643H50.427V0H9.56916V5.42643H15.2909V19.3117L0.711306 57.0494C0.487865 57.6399 0.368164 58.2623 0.368164 58.8928C0.368164 61.7097 2.65844 64 5.4754 64H54.5208C55.1512 64 55.7736 63.8803 56.3642 63.6569C58.9976 62.6434 60.3063 59.6828 59.2849 57.0494ZM20.7173 20.3252V5.58604H39.2789V20.3252L46.5327 39.1022C44.8809 38.6793 43.1731 38.4638 41.4335 38.4638C36.5497 38.4638 31.9213 40.1796 28.2425 43.2519C25.5273 45.52 22.1005 46.7602 18.5627 46.7551C15.9532 46.7551 13.4475 46.0928 11.237 44.8638L20.7173 20.3252ZM5.93824 58.5736L9.26592 49.9711C12.1148 51.4155 15.2829 52.1895 18.5707 52.1895C23.4544 52.1895 28.0829 50.4738 31.7617 47.4015C34.4669 45.1511 37.8505 43.8983 41.4415 43.8983C44.2345 43.8983 46.8998 44.6564 49.23 46.0609L54.0579 58.5736H5.93824Z"
          fill={foreground}
          stroke={background}
          strokeWidth="4"
        />
      </g>
      <defs>
        <clipPath id="clip0_66_8285">
          <rect fill={foreground} height="1024" transform="translate(-690 -407)" width="1440" />
        </clipPath>
      </defs>
    </svg>
  );
};

const ExceptionMessage: React.FC<ExceptionMessageProps> = ({
  ImageComponent,
  message,
  title,
  children,
}) => {
  const { ui } = useUI();

  return (
    <div className={css.base}>
      {ImageComponent && <ImageComponent darkLight={ui.darkLight} />}
      <div className={css.title}>{title}</div>
      {message && <span className={css.message}>{message}</span>}
      {children}
    </div>
  );
};

export const NoExperiments: React.FC = () => {
  return (
    <ExceptionMessage
      ImageComponent={ExperimentImage}
      message="Keep track of experiments you run in a project by connecting up your code."
      title="No Experiments">
      <Link external path={paths.docs('/post-training/model-registry.html')}>
        Quick Start Guide
      </Link>
    </ExceptionMessage>
  );
};

export const NoMatches: React.FC<{ clearFilters?: () => void }> = ({ clearFilters }) => (
  <ExceptionMessage title="No Matching Results">
    <Button onClick={clearFilters}>Clear Filters</Button>
  </ExceptionMessage>
);
export const Error: React.FC<{ fetchExperiments?: () => void }> = ({ fetchExperiments }) => (
  <ExceptionMessage ImageComponent={ImageAlert} title="Failed to Load Data">
    <Button onClick={fetchExperiments}>Retry</Button>
  </ExceptionMessage>
);
