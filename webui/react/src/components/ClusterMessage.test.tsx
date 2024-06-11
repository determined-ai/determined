// components/Counter.test.tsx
import { render, screen } from '@testing-library/react';

import { ClusterMessage } from 'stores/determinedInfo';
import ClusterMessageBanner from './ClusterMessage';

const setUp = (msg?: ClusterMessage) => {
  render(<ClusterMessageBanner message={msg} />); // render arbitrary components
};

const msg = 'this is a test msg for use in the ClusterMessageBanner test';

describe('ClusterMessageBanner', () => {
  it('should have a banner with cluster message text when there is a cluster message', () => {
    const time = new Date();
    const testMsg = { createdTime: time, endTime: time, message: msg, startTime: time };

    setUp(testMsg);
    // make sure these components exist.
    expect(screen.getByTestId('admin-msg')).toBeInTheDocument();
    expect(screen.getByTestId('cluster-msg')).toBeInTheDocument();

    // make sure the cluster message is visible, but the admin msg is not.
    expect(screen.getByText(msg)).toBeInTheDocument();
    expect(screen.getByTestId('admin-msg')).toBeInTheDocument();
    expect(screen.getByTestId('cluster-msg').textContent).toEqual(msg);
  });

  it('should not have a banner when there is no cluster message', () => {
    setUp();

    // setting a null message means nothing should show up in the ui.
    expect(screen.queryByTestId('admin-msg')).not.toBeInTheDocument();
    expect(screen.queryByText('cluster-msg')).not.toBeInTheDocument();
    expect(screen.queryByText('test msg')).not.toBeInTheDocument();
    expect(screen.queryByText('Message from Admin')).not.toBeInTheDocument();
  });
});
