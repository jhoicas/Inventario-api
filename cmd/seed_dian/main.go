// seed_dian genera scripts SQL para poblar tablas paramétricas DIAN (departamentos y municipios)
// a partir del XML oficial Municipios.xml.
//
// Uso: go run ./cmd/seed_dian [ruta/Municipios.xml]
// Por defecto busca Municipios.xml en el directorio actual.
// Escribe: internal/infrastructure/postgres/migrations/011_seed_locations.sql
package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type parametros struct {
	Tabla struct {
		Valores []valor `xml:"valor"`
	} `xml:"tabla"`
}

type valor struct {
	Cod    string `xml:"cod,attr"`
	Nombre string `xml:"nombre,attr"`
	Otro   struct {
		Codigo string `xml:"codigo,attr"`
		Valor  string `xml:"valor,attr"`
	} `xml:"otro"`
}

func main() {
	xmlPath := "Municipios.xml"
	if len(os.Args) > 1 {
		xmlPath = os.Args[1]
	}
	f, err := os.Open(xmlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Abrir XML: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	var p parametros
	dec := xml.NewDecoder(f)
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		if strings.EqualFold(charset, "ISO-8859-1") || strings.EqualFold(charset, "ISO8859-1") {
			return transform.NewReader(input, charmap.ISO8859_1.NewDecoder()), nil
		}
		return input, nil
	}
	if err := dec.Decode(&p); err != nil {
		fmt.Fprintf(os.Stderr, "Decodificar XML: %v\n", err)
		os.Exit(1)
	}

	// Departamentos únicos: (codigo, valor)
	deptMap := make(map[string]string)
	var cities []struct{ cod, nombre, deptCode string }
	for _, v := range p.Tabla.Valores {
		if v.Cod == "" || v.Nombre == "" || v.Otro.Codigo == "" || v.Otro.Valor == "" {
			continue
		}
		deptMap[v.Otro.Codigo] = v.Otro.Valor
		cities = append(cities, struct{ cod, nombre, deptCode string }{
			cod:      strings.TrimSpace(v.Cod),
			nombre:   strings.TrimSpace(v.Nombre),
			deptCode: strings.TrimSpace(v.Otro.Codigo),
		})
	}

	// Ordenar departamentos por código para salida estable
	var deptCodes []string
	for c := range deptMap {
		deptCodes = append(deptCodes, c)
	}
	sort.Strings(deptCodes)

	// Ruta del script de salida (relativa al módulo)
	moduleRoot := findModuleRoot()
	outPath := filepath.Join(moduleRoot, "internal", "infrastructure", "postgres", "migrations", "011_seed_locations.sql")
	out, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Crear archivo: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	// Escribir SQL: departamentos primero
	out.WriteString("-- Departamentos y municipios Colombia (código DANE)\n")
	out.WriteString("-- Generado desde Municipios.xml (DIAN)\n\n")

	out.WriteString("-- 1. Departamentos\n")
	out.WriteString("INSERT INTO locations_departments (code, name) VALUES\n")
	for i, c := range deptCodes {
		name := escapeSQL(deptMap[c])
		if i < len(deptCodes)-1 {
			fmt.Fprintf(out, "  ('%s', '%s'),\n", c, name)
		} else {
			fmt.Fprintf(out, "  ('%s', '%s')\n", c, name)
		}
	}
	out.WriteString("ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name;\n\n")

	// 2. Ciudades (municipios) con subquery al departamento
	out.WriteString("-- 2. Municipios (código DANE completo)\n")
	for _, city := range cities {
		name := escapeSQL(city.nombre)
		fmt.Fprintf(out, "INSERT INTO locations_cities (department_id, code, name)\n")
		fmt.Fprintf(out, "SELECT id, '%s', '%s' FROM locations_departments WHERE code = '%s'\n",
			city.cod, name, city.deptCode)
		out.WriteString("ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name;\n")
	}

	fmt.Printf("Generado %s: %d departamentos, %d municipios\n", outPath, len(deptCodes), len(cities))
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func findModuleRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}
