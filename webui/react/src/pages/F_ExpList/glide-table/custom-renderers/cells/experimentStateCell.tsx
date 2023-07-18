import { CustomCell, CustomRenderer, GridCellKind } from '@hpe.com/glide-data-grid';

import { CompoundRunState, JobState, RunState } from 'types';
import { Theme } from 'utils/themes';

import { roundedRect } from '../utils';

interface ExperimentStateCellProps {
  readonly appTheme: Theme;
  readonly kind: 'experiment-state-cell';
  readonly state: CompoundRunState;
}

const PI = Math.PI;
const QUADRANT = PI / 2;

export type ExperimentStateCell = CustomCell<ExperimentStateCellProps>;

const renderer: CustomRenderer<ExperimentStateCell> = {
  draw: (args, cell) => {
    const { ctx, rect, theme, requestAnimationFrame } = args;
    const { state, appTheme } = cell.data;

    const xPad = theme.cellHorizontalPadding;

    const radius = Math.min(12, rect.height / 2 - theme.cellVerticalPadding);
    const r = radius * 0.65;

    const drawX = rect.x + xPad;

    const x = drawX + r;
    const y = rect.y + 0.5 * rect.height;

    ctx.save();

    switch (state) {
      case JobState.SCHEDULED:
      case JobState.SCHEDULEDBACKFILLED:
      case JobState.QUEUED:
      case RunState.Queued: {
        const innnerCircleFill = appTheme.stageBorderWeak;
        const outerCircleFill = appTheme.stageStrong;
        const growth = 0.3 * r;
        const innerRadius = 0.4 * r;
        const outerRadius = r;
        const periodInMilliseconds = 2000;

        const ratio = (window.performance.now() % periodInMilliseconds) / periodInMilliseconds;
        const theta = 2 * PI * ratio;

        const progress = Math.sin(theta);

        const r0 = Math.max(innerRadius + growth * progress, 0);
        const r1 = outerRadius - 0.15 * growth * progress;

        ctx.beginPath();
        ctx.arc(x, y, Math.min(12, r1), 0, 2 * PI);
        ctx.fillStyle = outerCircleFill;
        ctx.fill();
        ctx.beginPath();
        ctx.arc(x, y, Math.min(12, r0), 0, 2 * PI);
        ctx.fillStyle = innnerCircleFill;
        ctx.fill();

        ctx.lineWidth = 1;
        break;
      }
      case RunState.Starting:
      case RunState.Pulling: {
        const darkSegmentStroke = appTheme.stageOn;
        const lightSegmentStroke = appTheme.stageBorderWeak;
        const periodInMilliseconds = 1000;
        const numFrames = 30;
        const ratio = (window.performance.now() % periodInMilliseconds) / periodInMilliseconds;
        const ratioTransform = Math.sin(ratio * QUADRANT);
        const roundedRatio = Math.round(numFrames * ratioTransform) / numFrames;
        const theta = PI * roundedRatio + PI / 4;
        ctx.lineWidth = 2;

        ctx.beginPath();
        ctx.strokeStyle = lightSegmentStroke;
        ctx.arc(x, y, r, 0, 2 * PI);
        ctx.stroke();
        ctx.beginPath();
        ctx.strokeStyle = darkSegmentStroke;
        ctx.arc(x, y, r, theta, theta + QUADRANT);
        ctx.stroke();
        ctx.beginPath();
        ctx.arc(x, y, r, PI + theta, theta + PI + QUADRANT);
        ctx.stroke();
        break;
      }
      case RunState.Running: {
        const progress = (window.performance.now() % 1000) / 1000;

        const startAngle = PI * 2 * progress;

        const gradient = ctx.createConicGradient(startAngle, x, y);
        gradient.addColorStop(0, '#00000000');
        gradient.addColorStop(0.25, '#00000000');
        gradient.addColorStop(1, appTheme.stageOnWeak);

        ctx.arc(x, y, Math.min(12, rect.height / 6), 0, 2 * PI);

        ctx.strokeStyle = gradient;
        ctx.lineWidth = 1.5;
        ctx.stroke();

        ctx.lineWidth = 1;
        break;
      }
      case RunState.Paused: {
        const barWidth = r * 0.6;
        const barHeight = r * 1.3;
        ctx.beginPath();
        roundedRect(ctx, x - r * 0.8, y - barHeight / 2, barWidth, barHeight, 1.5);
        roundedRect(ctx, x + r * 0.2, y - barHeight / 2, barWidth, barHeight, 1.5);
        ctx.fillStyle = appTheme.stageOn;
        ctx.fill();
        break;
      }
      case RunState.Completed: {
        const x0 = x - 0.3 * r;
        const r0 = 0.85 * r;
        ctx.beginPath();
        ctx.moveTo(x0 - 0.6 * r, y);
        ctx.lineTo(x0, y + r * 0.6);
        ctx.lineTo(x0 + r0 * 1.5, y - r * 0.6);
        ctx.lineWidth = 2;
        ctx.strokeStyle = appTheme.statusSuccess;
        ctx.stroke();
        break;
      }
      case RunState.Error:
      case RunState.Deleted:
      case RunState.Deleting:
      case RunState.DeleteFailed: {
        const k = 0.4;
        ctx.beginPath();
        ctx.arc(x, y, r, 0, 2 * PI);
        ctx.lineCap = 'round';
        ctx.moveTo(x - r * k, y - r * k);
        ctx.lineTo(x + r * k, y + r * k);
        ctx.moveTo(x + r * k, y - r * k);
        ctx.lineTo(x - r * k, y + r * k);
        ctx.lineWidth = 1.3;
        ctx.strokeStyle = appTheme.statusError;
        ctx.stroke();
        break;
      }
      case RunState.Active:
      case RunState.Unspecified:
      case JobState.UNSPECIFIED: {
        const periodInMilliseconds = 3000;
        const numFrames = 60;
        const ratio = (window.performance.now() % periodInMilliseconds) / periodInMilliseconds;
        const t = Math.round(PI * numFrames * ratio) / numFrames;
        const alpha1 = 0.3 + 0.7 * Math.abs(Math.sin(t)) ** 2;
        const alpha2 = 0.3 + 0.7 * Math.abs(Math.sin(t + Math.PI / 3)) ** 2;

        const r0 = r / 6;
        // Draw left dot

        ctx.fillStyle = appTheme.stageOn;
        ctx.globalAlpha = alpha1;
        ctx.beginPath();
        ctx.arc(x - 0.7 * r, y, r0, 0, 2 * Math.PI);
        ctx.fill();

        // Draw center dot
        ctx.beginPath();
        ctx.globalAlpha = alpha2;
        ctx.arc(x, y, r0, 0, 2 * Math.PI);
        ctx.fill();

        // Draw right dot
        ctx.beginPath();
        ctx.globalAlpha = alpha2;
        ctx.arc(x + 0.7 * r, y, r0, 0, 2 * Math.PI);
        ctx.fill();
        break;
      }
      default: {
        ctx.beginPath();
        ctx.arc(x, y, r, 0, 2 * PI);
        ctx.moveTo(x - r * 0.7, y - r * 0.7);
        ctx.lineTo(x + r * 0.7, y + r * 0.7);
        ctx.lineWidth = 1.3;
        ctx.strokeStyle = appTheme.stageOn;
        ctx.stroke();
        break;
      }
    }
    requestAnimationFrame();

    ctx.restore();

    return true;
  },
  isMatch: (cell: CustomCell): cell is ExperimentStateCell =>
    (cell.data as ExperimentStateCellProps).kind === 'experiment-state-cell',
  kind: GridCellKind.Custom,
  measure: () => 60,
  provideEditor: () => undefined,
};

export default renderer;
