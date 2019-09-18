package main

import "testing"

func Test_formatFileSize(t *testing.T) {
	type args struct {
		size int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "> 10G",
			args: args{size: 10_000_000_000},
			want: "10G",
		},
		{
			name: "> 1G",
			args: args{size: 1_100_000_000},
			want: "1.1G",
		},
		{
			name: "> 10M",
			args: args{size: 10_000_000},
			want: "10M",
		},
		{
			name: "> 1M",
			args: args{size: 1_100_000},
			want: "1.1M",
		},
		{
			name: "> 10K",
			args: args{size: 10_000},
			want: "10K",
		},
		{
			name: "> 1K",
			args: args{size: 1_100},
			want: "1.1K",
		},
		{
			name: "< 1K",
			args: args{size: 100},
			want: "100",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatFileSize(tt.args.size); got != tt.want {
				t.Errorf("formatFileSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
