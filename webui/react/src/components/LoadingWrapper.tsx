import { Skeleton, SkeletonProps } from 'antd';
import React, { useMemo } from 'react';

import Message, { Props as MessageProps, MessageType } from 'shared/components/Message';
import { isObject, validateEnum } from 'shared/utils/data';

import css from './LoadingWrapper.module.scss';

export enum LoadingState {
  Empty = 'Empty',
  Error = 'Error',
  Loaded = 'Loaded',
  Loading = 'Loading',
}

type LoadingMessageProps = React.ReactNode | MessageProps;
type LoadingSkeletonProps = React.ReactNode | SkeletonProps;
type LoadingStateConditions = {
  isEmpty?: boolean,
  isError?: boolean,
  isLoading?: boolean,
};

interface StyleProps {
  maxHeight?: boolean;
  noPadding?: boolean;
}

interface Props extends StyleProps {
  children: React.ReactNode;
  empty?: LoadingMessageProps;
  error?: LoadingMessageProps;
  loaded?: () => React.ReactNode;
  skeleton?: LoadingSkeletonProps;
  state: LoadingState | LoadingStateConditions;
}

const isLoadingStateConditions = (props: unknown): props is LoadingStateConditions => {
  return isObject(props);
};

const renderMessage = (props: LoadingMessageProps, state: LoadingState) => {
  if (React.isValidElement(props)) return <>{props}</>;
  const defaultProps: MessageProps = {
    title: state === LoadingState.Error ? 'Unable to load data' : 'No data to display',
    type: state === LoadingState.Error ? MessageType.Alert : MessageType.Empty,
    ...props as Partial<MessageProps>,
  };
  return <Message {...defaultProps} />;
};

const renderSkeleton = (props: LoadingSkeletonProps, styleProps: StyleProps) => {
  const classes = [ css.skeleton ];
  const content = React.isValidElement(props) ? props : <Skeleton {...props as SkeletonProps} />;

  if (styleProps.maxHeight) classes.push(css.maxHeight);
  if (styleProps.noPadding) classes.push(css.noPadding);

  return <div className={classes.join(' ')}>{content}</div>;
};

const LoadingWrapper: React.FC<Props> = (props: Props) => {
  const styleProps = useMemo(() => ({
    maxHeight: props.maxHeight,
    noPadding: props.noPadding,
  }), [ props.maxHeight, props.noPadding ]);

  const state = useMemo(() => {
    if (validateEnum(LoadingState, props.state)) return props.state as LoadingState;
    if (isLoadingStateConditions(props.state)) {
      if (props.state.isEmpty) return LoadingState.Empty;
      if (props.state.isError) return LoadingState.Error;
      if (props.state.isLoading) return LoadingState.Loading;
    }
    return LoadingState.Loaded;
  }, [ props.state ]);

  const jsx = useMemo(() => {
    if (state === LoadingState.Loaded) return <>{props.children}</>;
    if (state === LoadingState.Empty) return renderMessage(props.empty, state);
    if (state === LoadingState.Error) return renderMessage(props.error, state);
    return renderSkeleton(props.skeleton, styleProps);
  }, [ props.children, props.empty, props.error, props.skeleton, state, styleProps ]);

  return jsx;
};

export default LoadingWrapper;
