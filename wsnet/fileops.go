package wsnet

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "io"
	"os"
	"path/filepath"
	"strings"
)

func readStr(r io.Reader) (string, error) {
    var length uint32
    if err := binary.Read(r, binary.BigEndian, &length); err != nil {
        return "", fmt.Errorf("failed to read string length: %w", err)
    }
    data := make([]byte, length)
    if _, err := io.ReadFull(r, data); err != nil {
        return "", fmt.Errorf("failed to read string data: %w", err)
    }
    return string(data), nil
}

func writeStr(w io.Writer, s string) error {
    length := uint32(len(s))
    if err := binary.Write(w, binary.BigEndian, length); err != nil {
        return fmt.Errorf("failed to write string length: %w", err)
    }
    if _, err := w.Write([]byte(s)); err != nil {
        return fmt.Errorf("failed to write string data: %w", err)
    }
    return nil
}

type WSNFileEntry struct {
    Root   string     // variable-length string
    Name   string     // variable-length string
    IsDir  bool       // single byte in the buffer (0x00 or 0x01)
    Size   uint64     // 8 bytes
    ATime  uint64     // 8 bytes
    MTime  uint64     // 8 bytes
    CTime  uint64     // 8 bytes
}

type WSNFileRead struct {
    Size   uint32   // 4-byte size
    Offset uint64   // 8-byte offset
}

type WSNFileData struct {
    Offset uint64   // 8-byte offset
	Data   []byte   // variable-length data
}

type WSNFileOpen struct {
    Path   string
	Mode   string
}

type WSNFileCopy struct {
    SrcPath string
    DstPath string
}

type WSNFileMove struct {
    SrcPath string
    DstPath string
}

type WSNFileRM struct {
    Path string
}

type WSNDirCopy struct {
    SrcPath string
    DstPath string
}

type WSNDirMove struct {
    SrcPath string
    DstPath string
}

type WSNDirLS struct {
    Path string
}

type WSNDirMK struct {
    Path string
}

type WSNDirRM struct {
    Path string
}


func WSNFileOpenFromBytes(b []byte) (*WSNFileOpen, error) {
    buf := bytes.NewReader(b)

    path, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read path: %w", err)
    }

    mode, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read mode: %w", err)
    }

    return &WSNFileOpen{
    	Path: path,
        Mode: mode,
    }, nil
}

func (w *WSNFileOpen) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := writeStr(&buf, w.Path); err != nil {
        return nil, fmt.Errorf("failed to write Path: %w", err)
    }
    if err := writeStr(&buf, w.Mode); err != nil {
        return nil, fmt.Errorf("failed to write Mode: %w", err)
    }
    return buf.Bytes(), nil
}


func WSNFileEntryFromBytes(data []byte) (*WSNFileEntry, error) {
    buf := bytes.NewReader(data)
    var entry WSNFileEntry

    root, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read root: %w", err)
    }
    entry.Root = root

    name, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read name: %w", err)
    }
    entry.Name = name

    flag := make([]byte, 1)
    if _, err := io.ReadFull(buf, flag); err != nil {
        return nil, fmt.Errorf("failed to read is_dir byte: %w", err)
    }
    entry.IsDir = (flag[0] != 0)

    if err := binary.Read(buf, binary.BigEndian, &entry.Size); err != nil {
        return nil, fmt.Errorf("failed to read size: %w", err)
    }

    if err := binary.Read(buf, binary.BigEndian, &entry.ATime); err != nil {
        return nil, fmt.Errorf("failed to read atime: %w", err)
    }

    if err := binary.Read(buf, binary.BigEndian, &entry.MTime); err != nil {
        return nil, fmt.Errorf("failed to read mtime: %w", err)
    }

    if err := binary.Read(buf, binary.BigEndian, &entry.CTime); err != nil {
        return nil, fmt.Errorf("failed to read ctime: %w", err)
    }

    return &entry, nil
}

func (w *WSNFileEntry) ToData() ([]byte, error) {
    var buf bytes.Buffer

    if err := writeStr(&buf, w.Root); err != nil {
        return nil, fmt.Errorf("failed to write root: %w", err)
    }

    if err := writeStr(&buf, w.Name); err != nil {
        return nil, fmt.Errorf("failed to write name: %w", err)
    }

    var dirByte byte
    if w.IsDir {
        dirByte = 0x01
    } else {
        dirByte = 0x00
    }
    if err := buf.WriteByte(dirByte); err != nil {
        return nil, fmt.Errorf("failed to write is_dir: %w", err)
    }


    if err := binary.Write(&buf, binary.BigEndian, w.Size); err != nil {
        return nil, fmt.Errorf("failed to write size: %w", err)
    }

    if err := binary.Write(&buf, binary.BigEndian, w.ATime); err != nil {
        return nil, fmt.Errorf("failed to write atime: %w", err)
    }

    if err := binary.Write(&buf, binary.BigEndian, w.MTime); err != nil {
        return nil, fmt.Errorf("failed to write mtime: %w", err)
    }

    if err := binary.Write(&buf, binary.BigEndian, w.CTime); err != nil {
        return nil, fmt.Errorf("failed to write ctime: %w", err)
    }

    return buf.Bytes(), nil
}

func WSNFileDataFromFileData(offset uint64, size uint32, file *os.File) (*WSNFileData, error) {
	var w WSNFileData
	w.Offset = offset

	data := make([]byte, size)
	file.Seek(int64(offset), 0)
	n, err := file.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var buf bytes.Buffer
	// 4 bytes size of the data
	if err := binary.Write(&buf, binary.BigEndian, uint32(n)); err != nil {	
		return nil, fmt.Errorf("failed to write size: %w", err)
	}
	// the data
	if _, err := buf.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write data: %w", err)
	}
	w.Data = buf.Bytes()
	return &w, nil
}

func WSNFileDataFromBytes(data []byte) (*WSNFileData, error) {
    buf := bytes.NewReader(data)

    var w WSNFileData

	// 3) Read 8-byte offset (big-endian, unsigned)
    if err := binary.Read(buf, binary.BigEndian, &w.Offset); err != nil {
        return nil, fmt.Errorf("failed to read offset: %w", err)
    }

    // 4) Read the rest of the data, but it has a 4-byte size at the beginning
	sizeBytes := make([]byte, 4)
	if _, err := io.ReadFull(buf, sizeBytes); err != nil {
		return nil, fmt.Errorf("failed to read size: %w", err)
	}
	size := binary.BigEndian.Uint32(sizeBytes)

	w.Data = make([]byte, size)
	if _, err := io.ReadFull(buf, w.Data); err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
    return &w, nil
}

func (w *WSNFileData) ToData() ([]byte, error) {
    var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, w.Offset); err != nil {
        return nil, fmt.Errorf("failed to write offset: %w", err)
    }

	if _, err := buf.Write(w.Data); err != nil {
		return nil, fmt.Errorf("failed to write data: %w", err)
	}
	    
    return buf.Bytes(), nil
}

func WSNFileReadFromBytes(data []byte) (*WSNFileRead, error) {
    buf := bytes.NewReader(data)

    var w WSNFileRead

    // 2) Read 4-byte size (big-endian, unsigned)
    if err := binary.Read(buf, binary.BigEndian, &w.Size); err != nil {
        return nil, fmt.Errorf("failed to read size: %w", err)
    }

    // 3) Read 8-byte offset (big-endian, unsigned)
    if err := binary.Read(buf, binary.BigEndian, &w.Offset); err != nil {
        return nil, fmt.Errorf("failed to read offset: %w", err)
    }

    return &w, nil
}

func (w *WSNFileRead) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := binary.Write(&buf, binary.BigEndian, w.Size); err != nil {
        return nil, fmt.Errorf("failed to write size: %w", err)
    }
    if err := binary.Write(&buf, binary.BigEndian, w.Offset); err != nil {
        return nil, fmt.Errorf("failed to write offset: %w", err)
    }
    return buf.Bytes(), nil
}

func WSNFileRMFromBytes(b []byte) (*WSNFileRM, error) {
    buf := bytes.NewReader(b)

    src, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read srcpath: %w", err)
    }

    return &WSNFileRM{
        Path: src,
    }, nil
}

func (w *WSNFileRM) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := writeStr(&buf, w.Path); err != nil {
        return nil, fmt.Errorf("failed to write srcpath: %w", err)
    }
    return buf.Bytes(), nil
}


func WSNFileMoveFromBytes(b []byte) (*WSNFileMove, error) {
    buf := bytes.NewReader(b)

    src, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read srcpath: %w", err)
    }

    dst, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read dstpath: %w", err)
    }

    return &WSNFileMove{
        SrcPath: src,
        DstPath: dst,
    }, nil
}

func (w *WSNFileMove) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := writeStr(&buf, w.SrcPath); err != nil {
        return nil, fmt.Errorf("failed to write srcpath: %w", err)
    }
    if err := writeStr(&buf, w.DstPath); err != nil {
        return nil, fmt.Errorf("failed to write dstpath: %w", err)
    }
    return buf.Bytes(), nil
}


func WSNFileCopyFromBytes(b []byte) (*WSNFileCopy, error) {
    buf := bytes.NewReader(b)

    src, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read srcpath: %w", err)
    }

    dst, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read dstpath: %w", err)
    }

    return &WSNFileCopy{
        SrcPath: src,
        DstPath: dst,
    }, nil
}

func (w *WSNFileCopy) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := writeStr(&buf, w.SrcPath); err != nil {
        return nil, fmt.Errorf("failed to write srcpath: %w", err)
    }
    if err := writeStr(&buf, w.DstPath); err != nil {
        return nil, fmt.Errorf("failed to write dstpath: %w", err)
    }
    return buf.Bytes(), nil
}


func WSNDirMoveFromBytes(b []byte) (*WSNDirMove, error) {
    buf := bytes.NewReader(b)

    src, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read srcpath: %w", err)
    }

    dst, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read dstpath: %w", err)
    }

    return &WSNDirMove{
        SrcPath: src,
        DstPath: dst,
    }, nil
}

func (w *WSNDirMove) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := writeStr(&buf, w.SrcPath); err != nil {
        return nil, fmt.Errorf("failed to write srcpath: %w", err)
    }
    if err := writeStr(&buf, w.DstPath); err != nil {
        return nil, fmt.Errorf("failed to write dstpath: %w", err)
    }
    return buf.Bytes(), nil
}

func WSNDirCopyFromBytes(b []byte) (*WSNDirCopy, error) {
    buf := bytes.NewReader(b)

    src, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read srcpath: %w", err)
    }

    dst, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read dstpath: %w", err)
    }

    return &WSNDirCopy{
        SrcPath: src,
        DstPath: dst,
    }, nil
}

func (w *WSNDirCopy) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := writeStr(&buf, w.SrcPath); err != nil {
        return nil, fmt.Errorf("failed to write srcpath: %w", err)
    }
    if err := writeStr(&buf, w.DstPath); err != nil {
        return nil, fmt.Errorf("failed to write dstpath: %w", err)
    }
    return buf.Bytes(), nil
}



func WSNDirLSFromBytes(b []byte) (*WSNDirLS, error) {
    buf := bytes.NewReader(b)

    src, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read srcpath: %w", err)
    }

    return &WSNDirLS{
        Path: src,
    }, nil
}

func (w *WSNDirLS) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := writeStr(&buf, w.Path); err != nil {
        return nil, fmt.Errorf("failed to write srcpath: %w", err)
    }
    return buf.Bytes(), nil
}

func WSNDirMKFromBytes(b []byte) (*WSNDirMK, error) {
    buf := bytes.NewReader(b)

    src, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read srcpath: %w", err)
    }

    return &WSNDirMK{
        Path: src,
    }, nil
}

func (w *WSNDirMK) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := writeStr(&buf, w.Path); err != nil {
        return nil, fmt.Errorf("failed to write srcpath: %w", err)
    }
    return buf.Bytes(), nil
}


func WSNDirRMFromBytes(b []byte) (*WSNDirRM, error) {
    buf := bytes.NewReader(b)

    src, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read srcpath: %w", err)
    }

    return &WSNDirRM{
        Path: src,
    }, nil
}

func (w *WSNDirRM) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := writeStr(&buf, w.Path); err != nil {
        return nil, fmt.Errorf("failed to write srcpath: %w", err)
    }
    return buf.Bytes(), nil
}

func CopyDirectory(src, dst string) error {
    // Ensure the source actually exists.
    info, err := os.Stat(src)
    if err != nil {
        return fmt.Errorf("failed to access source directory %q: %v", src, err)
    }
    if !info.IsDir() {
        return fmt.Errorf("source %q is not a directory", src)
    }

    // Create the destination directory if it doesn't exist.
    if err := os.MkdirAll(dst, info.Mode()); err != nil {
        return fmt.Errorf("failed to create destination directory %q: %v", dst, err)
    }

    // Walk over the source directory tree.
    return filepath.Walk(src, func(path string, f os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // Compute the full destination path.
        relPath, err := filepath.Rel(src, path)
        if err != nil {
            return err
        }
        targetPath := filepath.Join(dst, relPath)

        if f.IsDir() {
            // Create subdirectories (preserve permissions).
            if err := os.MkdirAll(targetPath, f.Mode()); err != nil {
                return fmt.Errorf("failed to create directory %q: %v", targetPath, err)
            }
        } else {
            // Copy file
            if err := copyFile(path, targetPath, f); err != nil {
                return err
            }
        }
        return nil
    })
}

func copyFile(srcFile, dstFile string, fi os.FileInfo) error {
    // Open the source file
    src, err := os.Open(srcFile)
    if err != nil {
        return fmt.Errorf("failed to open %q: %w", srcFile, err)
    }
    defer src.Close()

    // Determine file mode
    var mode os.FileMode = 0o644
    if fi != nil {
        // Use the provided FileInfo for mode
        mode = fi.Mode()
    } else {
        // Try to stat the source file
        if statInfo, err := os.Stat(srcFile); err == nil {
            mode = statInfo.Mode()
        } else {
            // If that fails, we stick with 0644
            // (you could decide to return an error instead)
        }
    }

    // Create/overwrite the destination file with the computed mode
    dst, err := os.OpenFile(dstFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
    if err != nil {
        return fmt.Errorf("failed to create %q: %w", dstFile, err)
    }
    defer dst.Close()

    // Perform the actual copy
    if _, err := io.Copy(dst, src); err != nil {
        return fmt.Errorf("failed to copy from %q to %q: %w", srcFile, dstFile, err)
    }

    return nil
}

func openFilePythonMode(name string, pyMode string) (*os.File, error) {
    // Strip out any 'b' or 't' characters (Python binary/text modes).
    modeStr := strings.ReplaceAll(pyMode, "b", "")
    modeStr = strings.ReplaceAll(modeStr, "t", "")

    var flags int
    switch modeStr {
    case "r": // read-only
        flags = os.O_RDONLY
    case "w": // write-only, create/truncate
        flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
    case "a": // write-only, create/append
        flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
    case "r+": // read/write
        flags = os.O_RDWR
    case "w+": // read/write, create/truncate
        flags = os.O_RDWR | os.O_CREATE | os.O_TRUNC
    case "a+": // read/write, create/append
        flags = os.O_RDWR | os.O_CREATE | os.O_APPEND
    case "x": // write-only, create exclusive
        flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
    case "x+": // read/write, create exclusive
        flags = os.O_RDWR | os.O_CREATE | os.O_EXCL
    default:
        // If there's an unrecognized mode, return an error.
        // You could also default to "r".
        return nil, fmt.Errorf("unrecognized file mode %q", pyMode)
    }

    // If it's read-only, we don't need a file permission. But for
    // new-file creation we typically supply something like 0o666.
    const defaultPerm = 0o666

    return os.OpenFile(name, flags, defaultPerm)
}