import type { Parameters } from "./parameters.ts";
import { surface, dimensions } from "../../surface.ts";
import { rgb } from "../../color/index.ts";
import { fragment } from "./shader.ts";
export function paint(ctx: CanvasRenderingContext2D, p: Parameters): void {
  const canvas = surface(ctx.canvas.width, ctx.canvas.height);
  const gl = canvas.getContext("webgl", {
    preserveDrawingBuffer: true,
    premultipliedAlpha: false,
  });
  if (!gl) throw new Error("Voronoi backgrounds require WebGL");
  const shaders: WebGLShader[] = [];
  const program = gl.createProgram();
  if (!program) throw new Error("Cannot allocate WebGL program");
  let buffer: WebGLBuffer | null = null;
  try {
    for (const [kind, source] of [
      [
        gl.VERTEX_SHADER,
        "attribute vec2 position;void main(){gl_Position=vec4(position,0.,1.);}",
      ],
      [gl.FRAGMENT_SHADER, fragment],
    ] as const) {
      const shader = gl.createShader(kind)!;
      shaders.push(shader);
      gl.shaderSource(shader, source);
      gl.compileShader(shader);
      if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS))
        throw new Error(
          gl.getShaderInfoLog(shader) || "Shader compilation failed",
        );
      gl.attachShader(program, shader);
    }
    gl.linkProgram(program);
    if (!gl.getProgramParameter(program, gl.LINK_STATUS))
      throw new Error(gl.getProgramInfoLog(program) || "Shader linking failed");
    gl.useProgram(program);
    buffer = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    gl.bufferData(
      gl.ARRAY_BUFFER,
      new Float32Array([-1, -1, 1, -1, -1, 1, 1, 1]),
      gl.STATIC_DRAW,
    );
    const position = gl.getAttribLocation(program, "position");
    gl.enableVertexAttribArray(position);
    gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0);
    const uniform = (name: string) => gl.getUniformLocation(program, name);
    gl.uniform2f(uniform("iResolution"), canvas.width, canvas.height);
    gl.uniform1f(uniform("iTime"), p.timestamp);
    gl.uniform1f(uniform("SIZE"), p.scale);
    gl.uniform1f(uniform("iAngle"), (p.angle_degrees * Math.PI) / 180);
    gl.uniform3fv(uniform("col1"), rgb(p.color_a));
    gl.uniform3fv(uniform("col2"), rgb(p.color_b));
    gl.viewport(0, 0, canvas.width, canvas.height);
    gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
    const size = dimensions(ctx);
    ctx.drawImage(canvas, 0, 0, size.width, size.height);
  } finally {
    gl.deleteBuffer(buffer);
    gl.deleteProgram(program);
    shaders.forEach((shader) => gl.deleteShader(shader));
    gl.getExtension("WEBGL_lose_context")?.loseContext();
  }
}
