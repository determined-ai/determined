import React from 'react';

type Props = {
  fallback: (e: Error) => React.ReactNode;
  children: React.ReactNode;
};

type State = { error: Error | null };

export class ErrorBoundary extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo): void {
    console.error(error, info.componentStack);
  }

  render(): React.ReactNode {
    if (this.state.error !== null) {
      return this.props.fallback(this.state.error);
    }

    return this.props.children;
  }
}
