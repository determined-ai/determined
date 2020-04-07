import tensorflow as tf


def visualize_conv_weights(filters, name):
    """Visualize use weights in convolution filters.
    Args:
        filters: tensor containing the weights [H,W,Cin,Cout]
        name: label for tensorboard
    Returns:
        image of all weight
    """
    with tf.name_scope("visualize_w_" + name):
        filters = tf.transpose(filters, (3, 2, 0, 1))  # [h, w, cin, cout] -> [cout, cin, h, w]
        filters = tf.unstack(filters)  # --> cout * [cin, h, w]
        filters = tf.concat(filters, 1)  # --> [cin, cout * h, w]
        filters = tf.unstack(filters)  # --> cin * [cout * h, w]
        filters = tf.concat(filters, 1)  # --> [cout * h, cin * w]
        filters = tf.expand_dims(filters, 0)
        filters = tf.expand_dims(filters, -1)

    tf.summary.image("visualize_w_" + name, filters)


def visualize_conv_activations(activation, name):
    """Visualize activations for convolution layers.
    Remarks:
        This tries to place all activations into a square.
    Args:
        activation: tensor with the activation [B,H,W,C]
        name: label for tensorboard
    Returns:
        image of almost all activations
    """
    import math

    with tf.name_scope("visualize_act_" + name):
        _, h, w, c = activation.get_shape().as_list()
        rows = []
        c_per_row = int(math.sqrt(c))
        for y in range(0, c - c_per_row, c_per_row):
            row = activation[:, :, :, y : y + c_per_row]  # [?, H, W, 32] --> [?, H, W, 5]
            cols = tf.unstack(row, axis=3)  # [?, H, W, 5] --> 5 * [?, H, W]
            row = tf.concat(cols, 1)
            rows.append(row)

        viz = tf.concat(rows, 2)
    tf.summary.image("visualize_act_" + name, tf.expand_dims(viz, -1))
