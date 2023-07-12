import {
  CustomCell,
  CustomRenderer,
  getMiddleCenterBias,
  GridCellKind,
  measureTextCached,
} from '@hpe.com/glide-data-grid';

interface UserProfileCellProps {
  readonly kind: 'user-profile-cell';
  readonly image: string;
  readonly initials: string;
  readonly tint: string;
  readonly name?: string;
}

export type UserProfileCell = CustomCell<UserProfileCellProps>;

const renderer: CustomRenderer<UserProfileCell> = {
  draw: (args, cell) => {
    const { ctx, rect, theme, imageLoader, col, row } = args;
    const { image, name, initials, tint } = cell.data;

    const xPad = theme.cellHorizontalPadding;

    const radius = Math.min(12, rect.height / 2 - theme.cellVerticalPadding);

    const drawX = rect.x + xPad;

    const imageResult = imageLoader.loadOrGetImage(image, col, row);

    ctx.save();
    ctx.beginPath();
    ctx.arc(drawX + radius, rect.y + rect.height / 2, radius, 0, Math.PI * 2);
    ctx.globalAlpha = 1;
    ctx.fillStyle = tint;
    ctx.fill();
    ctx.fillStyle = '#FCFCFC';

    ctx.globalAlpha = 1;

    ctx.font = `600 10px ${theme.fontFamily}`;
    const metrics = measureTextCached(initials.slice(0, 2), ctx);
    ctx.fillText(
      initials.slice(0, 2),
      drawX + radius - metrics.width / 2,
      rect.y + rect.height / 2 + getMiddleCenterBias(ctx, `600 16px ${theme.fontFamily}`),
    );

    if (imageResult !== undefined) {
      ctx.save();
      ctx.beginPath();
      ctx.arc(drawX + radius, rect.y + rect.height / 2, radius, 0, Math.PI * 2);
      ctx.clip();

      ctx.drawImage(imageResult, drawX, rect.y + rect.height / 2 - radius, radius * 2, radius * 2);

      ctx.restore();
    }

    if (name !== undefined) {
      ctx.font = `${theme.baseFontStyle} ${theme.fontFamily}`;
      ctx.fillStyle = theme.textDark;
      ctx.fillText(
        name,
        drawX + radius * 2 + xPad,
        rect.y + rect.height / 2 + getMiddleCenterBias(ctx, theme),
      );
    }

    ctx.restore();

    return true;
  },
  isMatch: (cell: CustomCell): cell is UserProfileCell =>
    (cell.data as UserProfileCellProps).kind === 'user-profile-cell',
  kind: GridCellKind.Custom,
  measure: () => 50,
  onPaste: (v, d) => ({
    ...d,
    name: v,
  }),
  provideEditor: () => undefined,
};

export default renderer;
