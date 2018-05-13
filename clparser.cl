

const __constant unsigned int hexval32[256]=
        {['0']= 0, ['1']= 1, ['2']= 2, ['3']= 3, ['4']= 4, ['5']= 5,
        ['6']= 6, ['7']= 7, ['8']= 8, ['9']= 9, ['a']= 0xA, ['A']= 0xA,
        ['b']= 0xB, ['B']= 0xB, ['c']=0xC, ['C']=0xC, ['d']= 0xD, ['D']= 0xD,
        ['e']= 0xE, ['E']= 0xE, ['f']= 0xF, ['F']= 0xF};

const __constant int elemSize=32;

/* const __constant int maxX=800;
 const __constant int maxY=800; */


int searchNonSpace(__global uchar *line,int offset);
int searchSpace(__global uchar *line,int offset);
ushort parsUint(__global uchar *m, int len);
uint parseHex3ToBGR(__global uchar *m);


inline int searchNonSpace(__global uchar *line,int offset) {
    int i=offset;
    while (i < elemSize &&  line[i] == ' ' && line[i]!=0 ) {
        i++;
    }
    return i;
}

inline int searchSpace(__global uchar *line,int offset) {

    while (offset < elemSize &&  line[offset] != ' ' && line[offset]!=0 ) {
           offset++;
    }
    return offset;
}


inline  ushort  parsUint(__global uchar *m, int len) {
	const ushort ZERO = 0x30;

	switch (len) {
	case 3: //assumed to be the  most likely case
		return  100*(m[0]-ZERO) +
				10*(m[1]-ZERO) +
				m[2]-ZERO;
	case 4:
		return  1000*(m[0]-ZERO) +
				100*(m[1]-ZERO) +
				10*(m[2]-ZERO) +
				m[3]-ZERO;
	case 2:
		return	10*(m[0]-ZERO) +
				m[1]-ZERO;
	case 1:  //least likely case
		return  m[0] - ZERO;
	}

	//todo increment error count
	return 0;
}

inline uint parseHex3ToBGR(__global uchar *m)  {
	//MUL version, compiles to shifts
	// BB GG RR
	//printf("%u %u %u %u\n",(uint)m[0],(uint)m[1],(uint)m[2],(uint)m[3],(uint)m[4],(uint)m[5]);

	return 0x100000*hexval32[m[4]] + 0x010000*hexval32[m[5]] + 0x001000*hexval32[m[2]] +
           0x000100*hexval32[m[3]] + 0x000010*hexval32[m[0]] + hexval32[m[1]];
}



kernel void clparser(global uchar *lines,  global ushort *x,  global ushort *y,  global uint *color) {
    int idx = get_global_id(0);

    __global uchar *line =&(lines[idx*elemSize]);

    int startOfX = searchNonSpace(line,3);
    int endOfX = searchSpace(line,startOfX+1);
    int x1=parsUint(&line[startOfX],endOfX-startOfX);

    x[idx]=x1;
   // printf("x=%d",x1);

    int startOfY =endOfX+1;
    int endOfY = searchSpace(line,startOfY);
    y[idx]=parsUint(&line[startOfY],endOfY-startOfY);


    int startOfC = endOfY+1;
    //TODO add case to handle Alpha and garbage at end of PX line
    color[idx]=parseHex3ToBGR(&(line[startOfC]));
}

/*
kernel void clparser(global uchar *lines,  global uint *pixel, global int W, global int H) {
    int idx = get_global_id(0);

    __global uchar *line =&(lines[idx*elemSize]);

    int startOfX = searchNonSpace(line,3);
    int endOfX = searchSpace(line,startOfX+1);
    int x1=parsUint(&line[startOfX],endOfX-startOfX);

    x[idx]=x1;
   // printf("x=%d",x1);

    int startOfY =endOfX+1;
    int endOfY = searchSpace(line,startOfY);
    y[idx]=parsUint(&line[startOfY],endOfY-startOfY);


    int startOfC = endOfY+1;
    //TODO add case to handle Alpha and garbage at end of PX line
    color[idx]=parseHex3ToBGR(&(line[startOfC]));
    int offset= y1 * 800
    pixels[]
}
offset := y*W + x
	offset2 := (W*H - offset) - 1
	(*pixels)[offset2] = color
	*/