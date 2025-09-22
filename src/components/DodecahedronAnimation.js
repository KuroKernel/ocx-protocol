import React, { useEffect, useRef } from 'react';

const OctahedronAnimation = () => {
  const canvasRef = useRef(null);
  const animationRef = useRef(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const gl = canvas.getContext('webgl2') || canvas.getContext('webgl');
    if (!gl) {
      console.error('WebGL not supported');
      return;
    }

    // Handle resize
    const handleResize = () => {
      const rect = canvas.getBoundingClientRect();
      const dpr = window.devicePixelRatio || 1;
      canvas.width = rect.width * dpr;
      canvas.height = rect.height * dpr;
      gl.viewport(0, 0, canvas.width, canvas.height);
    };

    handleResize();
    window.addEventListener('resize', handleResize);

    // Vertex shader source
    const vertexShaderSource = `
      attribute vec3 position;
      attribute vec3 normal;
      attribute vec2 texCoord;
      uniform mat4 modelViewMatrix;
      uniform mat4 projectionMatrix;
      uniform mat3 normalMatrix;
      varying vec3 vNormal;
      varying vec3 vPosition;
      varying vec2 vTexCoord;
      
      void main() {
        vNormal = normalize(normalMatrix * normal);
        vPosition = position;
        vTexCoord = texCoord;
        gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
      }
    `;

    // Fragment shader source
    const fragmentShaderSource = `
      precision highp float;
      varying vec3 vNormal;
      varying vec3 vPosition;
      varying vec2 vTexCoord;
      uniform vec3 uColor;
      uniform vec3 uLightDirection;
      uniform float uTime;
      uniform float uAlpha;
      uniform bool uIsWireframe;
      
      void main() {
        vec3 normal = normalize(vNormal);
        float lightIntensity = max(dot(normal, uLightDirection), 0.0);
        
        float ambient = 0.4;
        lightIntensity = ambient + lightIntensity * 0.6;
        
        vec3 color = uColor * lightIntensity;
        
        if (uIsWireframe) {
          // Wireframe effect
          float edge = 1.0 - smoothstep(0.0, 0.02, min(vTexCoord.x, vTexCoord.y));
          color = mix(color, vec3(0.8, 0.8, 0.9), edge);
        }
        
        gl_FragColor = vec4(color, uAlpha);
      }
    `;

    // Create shader function
    function createShader(gl, type, source) {
      const shader = gl.createShader(type);
      gl.shaderSource(shader, source);
      gl.compileShader(shader);
      
      if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
        console.error('Shader compilation error:', gl.getShaderInfoLog(shader));
        gl.deleteShader(shader);
        return null;
      }
      
      return shader;
    }

    // Create program function
    function createProgram(gl, vertexShader, fragmentShader) {
      const program = gl.createProgram();
      gl.attachShader(program, vertexShader);
      gl.attachShader(program, fragmentShader);
      gl.linkProgram(program);
      
      if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
        console.error('Program linking error:', gl.getProgramInfoLog(program));
        gl.deleteProgram(program);
        return null;
      }
      
      return program;
    }

    // Generate octahedron vertices
    function generateOctahedronVertices() {
      const vertices = [
        [1, 0, 0], [-1, 0, 0],  // X axis
        [0, 1, 0], [0, -1, 0],  // Y axis  
        [0, 0, 1], [0, 0, -1]   // Z axis
      ];
      return vertices;
    }

    // Generate octahedron faces (8 triangular faces)
    function generateOctahedronFaces() {
      return [
        [0, 2, 4], [0, 4, 3], [0, 3, 5], [0, 5, 2],  // Top half
        [1, 4, 2], [1, 3, 4], [1, 5, 3], [1, 2, 5]   // Bottom half
      ];
    }

    // Generate internal pipe connections
    function generateInternalPipes() {
      const vertices = generateOctahedronVertices();
      const pipes = [];
      
      // Connect opposite vertices through center
      const connections = [
        [0, 1], // X axis
        [2, 3], // Y axis
        [4, 5]  // Z axis
      ];
      
      connections.forEach(([start, end]) => {
        const startPos = vertices[start];
        const endPos = vertices[end];
        pipes.push({
          start: startPos,
          end: endPos,
          radius: 0.15
        });
      });
      
      // Add diagonal connections
      const diagonals = [
        [0, 2], [0, 3], [0, 4], [0, 5],
        [1, 2], [1, 3], [1, 4], [1, 5],
        [2, 4], [2, 5], [3, 4], [3, 5]
      ];
      
      diagonals.forEach(([start, end]) => {
        const startPos = vertices[start];
        const endPos = vertices[end];
        pipes.push({
          start: startPos,
          end: endPos,
          radius: 0.08
        });
      });
      
      return pipes;
    }

    // Generate octahedron data
    function generateOctahedronData() {
      const vertices = generateOctahedronVertices();
      const faces = generateOctahedronFaces();
      
      const positions = [];
      const normals = [];
      const texCoords = [];
      const indices = [];
      
      faces.forEach((face) => {
        const faceVertices = face.map(i => vertices[i]);
        
        // Calculate face normal
        const v1 = [faceVertices[1][0] - faceVertices[0][0], faceVertices[1][1] - faceVertices[0][1], faceVertices[1][2] - faceVertices[0][2]];
        const v2 = [faceVertices[2][0] - faceVertices[0][0], faceVertices[2][1] - faceVertices[0][1], faceVertices[2][2] - faceVertices[0][2]];
        const normal = [
          v1[1] * v2[2] - v1[2] * v2[1],
          v1[2] * v2[0] - v1[0] * v2[2],
          v1[0] * v2[1] - v1[1] * v2[0]
        ];
        const normalLength = Math.sqrt(normal[0] * normal[0] + normal[1] * normal[1] + normal[2] * normal[2]);
        const normalizedNormal = [normal[0] / normalLength, normal[1] / normalLength, normal[2] / normalLength];
        
        const baseIndex = positions.length / 3;
        
        // Triangle vertices
        faceVertices.forEach((vertex, i) => {
          positions.push(...vertex);
          normals.push(...normalizedNormal);
          texCoords.push(i === 0 ? 0 : 1, i === 1 ? 0 : 1);
        });
        
        indices.push(baseIndex, baseIndex + 1, baseIndex + 2);
      });
      
      return { positions, normals, texCoords, indices };
    }

    // Generate pipe data
    function generatePipeData(pipe) {
      const segments = 16;
      const positions = [];
      const normals = [];
      const texCoords = [];
      const indices = [];
      
      const start = pipe.start;
      const end = pipe.end;
      const radius = pipe.radius;
      
      // Create cylinder between start and end
      const direction = [end[0] - start[0], end[1] - start[1], end[2] - start[2]];
      const length = Math.sqrt(direction[0] * direction[0] + direction[1] * direction[1] + direction[2] * direction[2]);
      const normalizedDirection = [direction[0] / length, direction[1] / length, direction[2] / length];
      
      // Create perpendicular vectors for cylinder cross-section
      let perp1, perp2;
      if (Math.abs(normalizedDirection[0]) < 0.9) {
        perp1 = [1, 0, 0];
      } else {
        perp1 = [0, 1, 0];
      }
      
      // Cross product to get perpendicular
      perp2 = [
        perp1[1] * normalizedDirection[2] - perp1[2] * normalizedDirection[1],
        perp1[2] * normalizedDirection[0] - perp1[0] * normalizedDirection[2],
        perp1[0] * normalizedDirection[1] - perp1[1] * normalizedDirection[0]
      ];
      
      const perp1Length = Math.sqrt(perp1[0] * perp1[0] + perp1[1] * perp1[1] + perp1[2] * perp1[2]);
      perp1 = [perp1[0] / perp1Length, perp1[1] / perp1Length, perp1[2] / perp1Length];
      
      const perp2Length = Math.sqrt(perp2[0] * perp2[0] + perp2[1] * perp2[1] + perp2[2] * perp2[2]);
      perp2 = [perp2[0] / perp2Length, perp2[1] / perp2Length, perp2[2] / perp2Length];
      
      for (let i = 0; i <= segments; i++) {
        const t = i / segments;
        const currentPos = [
          start[0] + direction[0] * t,
          start[1] + direction[1] * t,
          start[2] + direction[2] * t
        ];
        
        for (let j = 0; j < segments; j++) {
          const angle = (j / segments) * 2 * Math.PI;
          const cos = Math.cos(angle);
          const sin = Math.sin(angle);
          
          const offset = [
            perp1[0] * cos * radius + perp2[0] * sin * radius,
            perp1[1] * cos * radius + perp2[1] * sin * radius,
            perp1[2] * cos * radius + perp2[2] * sin * radius
          ];
          
          positions.push(
            currentPos[0] + offset[0],
            currentPos[1] + offset[1],
            currentPos[2] + offset[2]
          );
          
          normals.push(offset[0] / radius, offset[1] / radius, offset[2] / radius);
          texCoords.push(t, j / segments);
        }
      }
      
      // Create indices for cylinder
      for (let i = 0; i < segments; i++) {
        for (let j = 0; j < segments; j++) {
          const current = i * segments + j;
          const next = i * segments + ((j + 1) % segments);
          const below = (i + 1) * segments + j;
          const belowNext = (i + 1) * segments + ((j + 1) % segments);
          
          indices.push(current, next, below);
          indices.push(next, belowNext, below);
        }
      }
      
      return { positions, normals, texCoords, indices };
    }

    // Generate ball vertices (simple sphere)
    function generateBallVertices(radius = 0.08, segments = 16) {
      const positions = [];
      const normals = [];
      const texCoords = [];
      const indices = [];
      
      for (let i = 0; i <= segments; i++) {
        const lat = Math.PI * i / segments;
        const y = Math.cos(lat) * radius;
        const r = Math.sin(lat) * radius;
        
        for (let j = 0; j <= segments; j++) {
          const lon = 2 * Math.PI * j / segments;
          const x = Math.cos(lon) * r;
          const z = Math.sin(lon) * r;
          
          positions.push(x, y, z);
          normals.push(x / radius, y / radius, z / radius);
          texCoords.push(j / segments, i / segments);
        }
      }
      
      for (let i = 0; i < segments; i++) {
        for (let j = 0; j < segments; j++) {
          const a = i * (segments + 1) + j;
          const b = a + segments + 1;
          
          indices.push(a, b, a + 1);
          indices.push(a + 1, b, b + 1);
        }
      }
      
      return { positions, normals, texCoords, indices };
    }

    // Matrix math functions
    function createMatrix() {
      return new Float32Array(16);
    }

    function identityMatrix(matrix) {
      matrix[0] = 1; matrix[1] = 0; matrix[2] = 0; matrix[3] = 0;
      matrix[4] = 0; matrix[5] = 1; matrix[6] = 0; matrix[7] = 0;
      matrix[8] = 0; matrix[9] = 0; matrix[10] = 1; matrix[11] = 0;
      matrix[12] = 0; matrix[13] = 0; matrix[14] = 0; matrix[15] = 1;
    }

    function perspectiveMatrix(matrix, fov, aspect, near, far) {
      const f = Math.tan(Math.PI * 0.5 - 0.5 * fov);
      const rangeInv = 1.0 / (near - far);
      
      matrix[0] = f / aspect; matrix[1] = 0; matrix[2] = 0; matrix[3] = 0;
      matrix[4] = 0; matrix[5] = f; matrix[6] = 0; matrix[7] = 0;
      matrix[8] = 0; matrix[9] = 0; matrix[10] = (near + far) * rangeInv; matrix[11] = -1;
      matrix[12] = 0; matrix[13] = 0; matrix[14] = near * far * rangeInv * 2; matrix[15] = 0;
    }

    function translateMatrix(matrix, x, y, z) {
      matrix[12] += matrix[0] * x + matrix[4] * y + matrix[8] * z;
      matrix[13] += matrix[1] * x + matrix[5] * y + matrix[9] * z;
      matrix[14] += matrix[2] * x + matrix[6] * y + matrix[10] * z;
      matrix[15] += matrix[3] * x + matrix[7] * y + matrix[11] * z;
    }

    function rotateXMatrix(matrix, angle) {
      const c = Math.cos(angle);
      const s = Math.sin(angle);
      const m1 = matrix[4], m2 = matrix[5], m3 = matrix[6], m4 = matrix[7];
      matrix[4] = m1 * c + matrix[8] * s;
      matrix[5] = m2 * c + matrix[9] * s;
      matrix[6] = m3 * c + matrix[10] * s;
      matrix[7] = m4 * c + matrix[11] * s;
      matrix[8] = m1 * -s + matrix[8] * c;
      matrix[9] = m2 * -s + matrix[9] * c;
      matrix[10] = m3 * -s + matrix[10] * c;
      matrix[11] = m4 * -s + matrix[11] * c;
    }

    function rotateYMatrix(matrix, angle) {
      const c = Math.cos(angle);
      const s = Math.sin(angle);
      const m0 = matrix[0], m1 = matrix[1], m2 = matrix[2], m3 = matrix[3];
      matrix[0] = m0 * c + matrix[8] * -s;
      matrix[1] = m1 * c + matrix[9] * -s;
      matrix[2] = m2 * c + matrix[10] * -s;
      matrix[3] = m3 * c + matrix[11] * -s;
      matrix[8] = m0 * s + matrix[8] * c;
      matrix[9] = m1 * s + matrix[9] * c;
      matrix[10] = m2 * s + matrix[10] * c;
      matrix[11] = m3 * s + matrix[11] * c;
    }

    // Initialize WebGL
    const vertexShader = createShader(gl, gl.VERTEX_SHADER, vertexShaderSource);
    const fragmentShader = createShader(gl, gl.FRAGMENT_SHADER, fragmentShaderSource);
    const program = createProgram(gl, vertexShader, fragmentShader);

    // Get attribute and uniform locations
    const positionLocation = gl.getAttribLocation(program, 'position');
    const normalLocation = gl.getAttribLocation(program, 'normal');
    const texCoordLocation = gl.getAttribLocation(program, 'texCoord');
    const modelViewMatrixLocation = gl.getUniformLocation(program, 'modelViewMatrix');
    const projectionMatrixLocation = gl.getUniformLocation(program, 'projectionMatrix');
    const normalMatrixLocation = gl.getUniformLocation(program, 'normalMatrix');
    const colorLocation = gl.getUniformLocation(program, 'uColor');
    const lightDirectionLocation = gl.getUniformLocation(program, 'uLightDirection');
    const timeLocation = gl.getUniformLocation(program, 'uTime');
    const alphaLocation = gl.getUniformLocation(program, 'uAlpha');
    const isWireframeLocation = gl.getUniformLocation(program, 'uIsWireframe');

    // Generate geometry
    const octahedronData = generateOctahedronData();
    const pipes = generateInternalPipes();
    const ballData = generateBallVertices();

    // Create buffers
    function createBuffer(data) {
      const buffer = gl.createBuffer();
      gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
      gl.bufferData(gl.ARRAY_BUFFER, new Float32Array(data), gl.STATIC_DRAW);
      return buffer;
    }

    function createIndexBuffer(data) {
      const buffer = gl.createBuffer();
      gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, buffer);
      gl.bufferData(gl.ELEMENT_ARRAY_BUFFER, new Uint16Array(data), gl.STATIC_DRAW);
      return buffer;
    }

    // Create octahedron buffers
    const octahedronPositionBuffer = createBuffer(octahedronData.positions);
    const octahedronNormalBuffer = createBuffer(octahedronData.normals);
    const octahedronTexCoordBuffer = createBuffer(octahedronData.texCoords);
    const octahedronIndexBuffer = createIndexBuffer(octahedronData.indices);

    // Create pipe buffers
    const pipeBuffers = pipes.map(pipe => {
      const pipeData = generatePipeData(pipe);
      return {
        position: createBuffer(pipeData.positions),
        normal: createBuffer(pipeData.normals),
        texCoord: createBuffer(pipeData.texCoords),
        index: createIndexBuffer(pipeData.indices)
      };
    });

    const ballPositionBuffer = createBuffer(ballData.positions);
    const ballNormalBuffer = createBuffer(ballData.normals);
    const ballTexCoordBuffer = createBuffer(ballData.texCoords);
    const ballIndexBuffer = createIndexBuffer(ballData.indices);

    // Set up matrices
    const projectionMatrix = createMatrix();
    const modelViewMatrix = createMatrix();
    const normalMatrix = createMatrix();

    // Animation variables
    let time = 0;
    const vertices = generateOctahedronVertices();
    const ballPositions = vertices;
    
    let currentVertexIndex = 0;
    let ballPosition = [...ballPositions[0]];
    let ballTarget = [...ballPositions[1]];
    let ballProgress = 0;

    // Render function
    function render() {
      time += 0.016; // ~60fps

      // Update ball position
      ballProgress += 0.03; // Medium to quick speed
      if (ballProgress >= 1) {
        ballProgress = 0;
        currentVertexIndex = (currentVertexIndex + 1) % ballPositions.length;
        ballTarget = [...ballPositions[currentVertexIndex]];
        ballPosition = [...ballPositions[currentVertexIndex === 0 ? ballPositions.length - 1 : currentVertexIndex - 1]];
      }

      // Interpolate ball position
      const currentBallPos = [
        ballPosition[0] + (ballTarget[0] - ballPosition[0]) * ballProgress,
        ballPosition[1] + (ballTarget[1] - ballPosition[1]) * ballProgress,
        ballPosition[2] + (ballTarget[2] - ballPosition[2]) * ballProgress
      ];

      // Clear canvas
      gl.clearColor(0.95, 0.95, 0.95, 1.0);
      gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);

      // Enable depth testing and blending
      gl.enable(gl.DEPTH_TEST);
      gl.enable(gl.BLEND);
      gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
      gl.depthFunc(gl.LEQUAL);

      // Use program
      gl.useProgram(program);

      // Set up projection matrix
      identityMatrix(projectionMatrix);
      perspectiveMatrix(projectionMatrix, Math.PI / 4, canvas.width / canvas.height, 0.1, 100);

      // Set up model view matrix
      identityMatrix(modelViewMatrix);
      translateMatrix(modelViewMatrix, 0, 0, -3);
      rotateYMatrix(modelViewMatrix, time * 0.1); // Slow rotation
      rotateXMatrix(modelViewMatrix, time * 0.05);

      // Set uniforms
      gl.uniformMatrix4fv(projectionMatrixLocation, false, projectionMatrix);
      gl.uniformMatrix4fv(modelViewMatrixLocation, false, modelViewMatrix);
      gl.uniformMatrix3fv(normalMatrixLocation, false, normalMatrix);
      gl.uniform3f(lightDirectionLocation, 0.5, 1, 0.5);
      gl.uniform1f(timeLocation, time);

      // Draw octahedron (wireframe)
      gl.uniform3f(colorLocation, 0.2, 0.2, 0.2); // Dark gray
      gl.uniform1f(alphaLocation, 0.8);
      gl.uniform1i(isWireframeLocation, true);

      gl.bindBuffer(gl.ARRAY_BUFFER, octahedronPositionBuffer);
      gl.enableVertexAttribArray(positionLocation);
      gl.vertexAttribPointer(positionLocation, 3, gl.FLOAT, false, 0, 0);

      gl.bindBuffer(gl.ARRAY_BUFFER, octahedronNormalBuffer);
      gl.enableVertexAttribArray(normalLocation);
      gl.vertexAttribPointer(normalLocation, 3, gl.FLOAT, false, 0, 0);

      gl.bindBuffer(gl.ARRAY_BUFFER, octahedronTexCoordBuffer);
      gl.enableVertexAttribArray(texCoordLocation);
      gl.vertexAttribPointer(texCoordLocation, 2, gl.FLOAT, false, 0, 0);

      gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, octahedronIndexBuffer);
      gl.drawElements(gl.TRIANGLES, octahedronData.indices.length, gl.UNSIGNED_SHORT, 0);

      // Draw pipes (transparent)
      gl.uniform1i(isWireframeLocation, false);
      gl.uniform3f(colorLocation, 0.1, 0.3, 0.6); // Blue pipes
      gl.uniform1f(alphaLocation, 0.6);

      pipeBuffers.forEach(pipeBuffer => {
        gl.bindBuffer(gl.ARRAY_BUFFER, pipeBuffer.position);
        gl.enableVertexAttribArray(positionLocation);
        gl.vertexAttribPointer(positionLocation, 3, gl.FLOAT, false, 0, 0);

        gl.bindBuffer(gl.ARRAY_BUFFER, pipeBuffer.normal);
        gl.enableVertexAttribArray(normalLocation);
        gl.vertexAttribPointer(normalLocation, 3, gl.FLOAT, false, 0, 0);

        gl.bindBuffer(gl.ARRAY_BUFFER, pipeBuffer.texCoord);
        gl.enableVertexAttribArray(texCoordLocation);
        gl.vertexAttribPointer(texCoordLocation, 2, gl.FLOAT, false, 0, 0);

        gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, pipeBuffer.index);
        gl.drawElements(gl.TRIANGLES, pipeBuffer.index.length, gl.UNSIGNED_SHORT, 0);
      });

      // Draw ball
      gl.uniform3f(colorLocation, 0.9, 0.1, 0.1); // Bright red ball
      gl.uniform1f(alphaLocation, 1.0);
      gl.uniform1i(isWireframeLocation, false);

      identityMatrix(modelViewMatrix);
      translateMatrix(modelViewMatrix, currentBallPos[0], currentBallPos[1], currentBallPos[2] - 3);
      rotateYMatrix(modelViewMatrix, time * 2);
      rotateXMatrix(modelViewMatrix, time * 1.5);

      gl.uniformMatrix4fv(modelViewMatrixLocation, false, modelViewMatrix);

      gl.bindBuffer(gl.ARRAY_BUFFER, ballPositionBuffer);
      gl.enableVertexAttribArray(positionLocation);
      gl.vertexAttribPointer(positionLocation, 3, gl.FLOAT, false, 0, 0);

      gl.bindBuffer(gl.ARRAY_BUFFER, ballNormalBuffer);
      gl.enableVertexAttribArray(normalLocation);
      gl.vertexAttribPointer(normalLocation, 3, gl.FLOAT, false, 0, 0);

      gl.bindBuffer(gl.ARRAY_BUFFER, ballTexCoordBuffer);
      gl.enableVertexAttribArray(texCoordLocation);
      gl.vertexAttribPointer(texCoordLocation, 2, gl.FLOAT, false, 0, 0);

      gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, ballIndexBuffer);
      gl.drawElements(gl.TRIANGLES, ballData.indices.length, gl.UNSIGNED_SHORT, 0);

      animationRef.current = requestAnimationFrame(render);
    }

    // Start animation
    render();

    // Cleanup
    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      style={{ 
        width: '100%', 
        height: '100%', 
        display: 'block',
        minHeight: '300px'
      }}
    />
  );
};

export default OctahedronAnimation;
